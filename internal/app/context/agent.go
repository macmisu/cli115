package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/deadblue/elevengo"
	"github.com/skip2/go-qrcode"
	"go.dead.blue/cli115/internal/app/config"
	"go.dead.blue/cli115/internal/pkg/util"
	"os"
	"path"
)

var (
	errUserCanceled  = errors.New("user canceled this login")
	errUnknownStatus = errors.New("unknow QRcode status")
)

type CookieData struct {
	Uid  string `json:"uid"`
	Cid  string `json:"cid"`
	Seid string `json:"seid"`
}

func initAgent(opts *config.Options) (agent *elevengo.Agent, err error) {
	agent = elevengo.Default()
	// try load cookie
	if cr, err := loadCookie(opts); err == nil {
		if err = agent.CredentialImport(cr); err == nil {
			return agent, nil
		}
	}
	// prompt user to login
	if err = login(agent); err == nil {
		_ = saveCookie(agent, opts)
	}
	return
}

func loadCookie(opts *config.Options) (cr *elevengo.Credential, err error) {
	// make credentials by arguments
	if opts.Uid != "" && opts.Cid != "" && opts.Seid != "" {
		cr = &elevengo.Credential{
			UID:  opts.Uid,
			CID:  opts.Cid,
			SEID: opts.Seid,
		}
		return
	}
	// load cookie from cookie file
	file, err := os.Open(opts.CookieFile)
	if err != nil {
		return
	}
	defer util.QuietlyClose(file)
	// decode cookie
	jd, data := json.NewDecoder(file), &CookieData{}
	if err = jd.Decode(data); err == nil {
		cr = &elevengo.Credential{
			UID:  data.Uid,
			CID:  data.Cid,
			SEID: data.Seid,
		}
	}
	return
}

func saveCookie(agent *elevengo.Agent, opts *config.Options) (err error) {
	// export credentials
	cr, err := agent.CredentialExport()
	if err != nil {
		return
	}
	// make directory
	if err = os.MkdirAll(path.Dir(opts.CookieFile), 0755); err != nil {
		return
	}
	// open cookie file for writing
	file, err := os.OpenFile(opts.CookieFile, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer util.QuietlyClose(file)
	// write to file
	je, data := json.NewEncoder(file), &CookieData{
		Uid:  cr.UID,
		Cid:  cr.CID,
		Seid: cr.SEID,
	}
	if err = je.Encode(data); err == nil {
		fmt.Printf("Cookie saved at: %s", opts.CookieFile)
	}
	return err
}

func login(agent *elevengo.Agent) (err error) {
	// retry when QRcode expired
	for {
		// Get QRcode
		session, err := agent.QrcodeStart()
		if err != nil {
			return err
		}
		code, err := qrcode.New(session.Content, qrcode.Medium)
		if err != nil {
			return err
		}
		fmt.Println("Please scan the QRcode on mobile App:")
		fmt.Print(code.ToSmallString(false))
		// Wait for login
		for wait := true; wait; {
			if status, err := agent.QrcodeStatus(session); err != nil {
				if elevengo.IsQrcodeExpire(err) {
					fmt.Println("QRcode expired, request a new one ...")
					wait = false
				} else {
					return err
				}
			} else {
				if status.IsAllowed() {
					return agent.QrcodeLogin(session)
				} else if status.IsCanceled() {
					return errUserCanceled
				} else if status.IsWaiting() || status.IsScanned() {
					fmt.Println("Waiting for scanning...")
				} else {
					return errUnknownStatus
				}
			}
		}
	}
	return nil
}
