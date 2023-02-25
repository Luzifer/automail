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
		StorageType    string        `flag:"storage-type" default:"file" description:"Driver to use for storing persistent info"`
		StorageDSN     string        `flag:"storage-dsn" default:"store.yaml" description:"Where to store persistent info"`
		VersionAndExit bool          `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func init() {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.WithError(err).Fatalf("parsing commandline options")
	}

	if cfg.VersionAndExit {
		fmt.Printf("automail %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("parsing log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	bodySection, err := imap.ParseBodySectionName("BODY[]")
	if err != nil {
		log.WithError(err).Fatal("parsing body section")
	}

	conf, err := loadConfig()
	if err != nil {
		log.WithError(err).Fatal("loading config")
	}

	store, err := newStorage(cfg.StorageType, cfg.StorageDSN)
	if err != nil {
		log.WithError(err).Fatal("creating storage interface")
	}

	if err = store.Load(); err != nil {
		log.WithError(err).Fatal("loading persistent storage data")
	}

	var (
		imapClient *client.Client
		messages   = make(chan *imap.Message, 1000)
		needLogin  = make(chan struct{}, 1)
		sigs       = make(chan os.Signal, 1)
		ticker     = time.NewTicker(cfg.FetchInterval)
	)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	needLogin <- struct{}{}

	for {
		select {

		case <-needLogin:
			if imapClient != nil {
				imapClient.Close()
			}

			imapClient, err = client.DialTLS(fmt.Sprintf("%s:%d", cfg.IMAPHost, cfg.IMAPPort), nil)
			if err != nil {
				log.WithError(err).Fatal("connecting to IMAP server")
			}

			if err = imapClient.Login(cfg.IMAPUser, cfg.IMAPPass); err != nil {
				log.WithError(err).Fatal("loggin in to IMAP server")
			}

			log.Info("IMAP connected and logged in")

			if _, err = imapClient.Select(cfg.Mailbox, false); err != nil {
				log.WithError(err).Fatal("selecting mailbox")
			}

			go func() {
				// Trigger re-login when log-out was received
				<-imapClient.LoggedOut()
				needLogin <- struct{}{}
			}()

		case <-ticker.C:
			if _, err := imapClient.Select(cfg.Mailbox, false); err != nil {
				log.WithError(err).Error("selecting mailbox")
				continue
			}

			seq, err := imap.ParseSeqSet(fmt.Sprintf("%d:*", store.GetLastUID()+1))
			if err != nil {
				log.WithError(err).Error("parsing sequence set")
				continue
			}

			ids, err := imapClient.UidSearch(&imap.SearchCriteria{
				Uid: seq,
			})
			if err != nil {
				log.WithError(err).Error("searching for messages")
				continue
			}

			if len(ids) == 0 {
				continue
			}

			tmpMsg := make(chan *imap.Message)
			go func() {
				for msg := range tmpMsg {
					if msg.Uid <= store.GetLastUID() {
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
				log.WithError(err).Error("fetching messages")
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
				log.WithError(err).Error("parsing message")
				continue
			}

			log.WithFields(log.Fields{
				"subject": mail.GetHeader("subject"),
				"uid":     msg.Uid,
			}).Debug("Fetched message")

			// Check all handlers whether they want to handle the message
			for _, hdl := range conf.Handlers {
				if hdl.Handles(mail) {
					go func(msg *imap.Message, hdl mailHandler) {
						if err := hdl.Process(imapClient, msg, mail); err != nil {
							log.WithError(err).Error("processing message")
						}
					}(msg, hdl)
				}
			}

			// Mark message as processed in store
			if msg.Uid > store.GetLastUID() {
				store.SetUID(msg.Uid)
				if err = store.Save(); err != nil {
					log.WithError(err).Error("saving storage")
				}
			}

		}
	}
}
