package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/jhillyerd/enmime"
	log "github.com/sirupsen/logrus"

	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		Config         string        `flag:"config,c" default:"config.yaml" description:"Configuration file with instruction"`
		FetchInterval  time.Duration `flag:"interval,i" default:"1m" description:"Interval to fetch mails"`
		IMAPHost       string        `flag:"imap-host,h" default:"" description:"Host of the IMAP server" validate:"nonzero"`
		IMAPPort       int           `flag:"imap-port" default:"993" description:"Port of the IMAP server" validate:"nonzero"`
		IMAPUser       string        `flag:"imap-user,u" default:"" description:"Username to access the IMAP server" validate:"nonzero"`
		IMAPPass       string        `flag:"imap-pass,p" default:"" description:"Password to access the IMAP server" validate:"nonzero"`
		LogLevel       string        `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		Mailbox        string        `flag:"mailbox,m" default:"INBOX" description:"Mailbox to fetch from"`
		StorageFile    string        `flag:"storage-file" default:"store.yaml" description:"Where to store persistent info"`
		VersionAndExit bool          `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func init() {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("automail %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("Unable to parse log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	bodySection, err := imap.ParseBodySectionName("BODY[]")
	if err != nil {
		log.WithError(err).Fatal("Unable to parse body section")
	}

	conf, err := loadConfig()
	if err != nil {
		log.WithError(err).Fatal("Unable to load config")
	}

	store, err := loadStorage()
	if err != nil {
		log.WithError(err).Fatal("Unable to load storage file")
	}

	imapClient, err := client.DialTLS(fmt.Sprintf("%s:%d", cfg.IMAPHost, cfg.IMAPPort), nil)
	if err != nil {
		log.WithError(err).Fatal("Unable to connect to IMAP server")
	}
	defer imapClient.Close()

	if err = imapClient.Login(cfg.IMAPUser, cfg.IMAPPass); err != nil {
		log.WithError(err).Fatal("Unable to login to IMAP server")
	}

	log.Info("IMAP connected and logged in")

	if _, err = imapClient.Select(cfg.Mailbox, false); err != nil {
		log.WithError(err).Fatal("Unable to select mailbox")
	}

	var (
		messages = make(chan *imap.Message, 1000)
		sigs     = make(chan os.Signal)
		ticker   = time.NewTicker(cfg.FetchInterval)
	)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	for {
		select {

		case <-ticker.C:
			seq, err := imap.ParseSeqSet(fmt.Sprintf("%d:*", store.LastUID+1))
			if err != nil {
				log.WithError(err).Error("Unable to parse sequence set")
				continue
			}

			ids, err := imapClient.UidSearch(&imap.SearchCriteria{
				Uid: seq,
			})
			if err != nil {
				log.WithError(err).Error("Unable to search for messages")
				continue
			}

			if len(ids) == 0 {
				continue
			}

			var tmpMsg = make(chan *imap.Message)
			go func() {
				for msg := range tmpMsg {
					if msg.Uid <= store.LastUID {
						continue
					}
					messages <- msg
				}
			}()

			fetchSeq := &imap.SeqSet{}
			fetchSeq.AddNum(ids...)

			if err = imapClient.UidFetch(fetchSeq, []imap.FetchItem{
				imap.FetchFlags,
				imap.FetchItem("BODY.PEEK[]"),
				imap.FetchUid,
			}, tmpMsg); err != nil {
				log.WithError(err).Error("Unable to fetch messages")
				continue
			}

		case <-sigs:
			return

		case msg := <-messages:
			body := msg.GetBody(bodySection)
			if body == nil {
				log.WithField("uid", msg.Uid).Debug("Got message with nil body")
				continue
			}

			mail, err := enmime.ReadEnvelope(body)
			if err != nil {
				log.WithError(err).Error("Unable to parse message")
				continue
			}

			log.WithFields(log.Fields{
				"subject": mail.GetHeader("subject"),
				"uid":     msg.Uid,
			}).Debug("Fetched message")

			// Check all handlers whether they want to handle the message
			for _, hdl := range conf.Handlers {
				if hdl.Handles(mail) {
					go func(msg *imap.Message) {
						if err := hdl.Process(imapClient, msg, mail); err != nil {
							log.WithError(err).Error("Error while processing message")
						}
					}(msg)
				}
			}

			// Mark message as processed in store
			if msg.Uid > store.LastUID {
				store.LastUID = msg.Uid
				if err = store.saveStorage(); err != nil {
					log.WithError(err).Error("Unable to save storage")
				}
			}

		}
	}
}
