package command

import (
	"fmt"
	"github.com/deadblue/elevengo"
	"go.dead.blue/cli115/context"
	"go.dead.blue/cli115/internal/app"
	"os"
	"os/exec"
)

type DlCommand struct {
	ArgsCommand
}

func (c *DlCommand) Name() string {
	return "dl"
}

func (c *DlCommand) ImplExec(ctx *context.Impl, args []string) (err error) {
	if len(args) == 0 {
		return errArgsNotEnough
	}
	// Find file
	file := ctx.Fs.File(args[0])
	if file == nil {
		return errFileNotExist
	}
	if !file.IsFile {
		return errNotFile
	}
	fmt.Printf("Downloading file: %s\n", file.Name)
	// Create download ticket
	ticket, err := ctx.Agent.CreateDownloadTicket(file.PickCode)
	if err != nil {
		return
	}
	// Prefer to use aria2 if available.
	if ctx.Conf.Aria2 != nil {
		return c.aria2Download(ctx.Conf.Aria2, ticket, file.Sha1)
	} else if ctx.Conf.Curl != nil {
		return c.curlDownload(ctx.Conf.Curl, ticket)
	} else {
		return errNoDownloader
	}
}

func (c *DlCommand) aria2Download(conf *app.Aria2Conf, ticket *elevengo.DownloadTicket, sha1 string) error {
	if !conf.Rpc {
		cmd := exec.Command(conf.Path,
			"--max-connection-per-server=2",
			"--split=16", "--min-split-size=1M",
			fmt.Sprintf("--out=%s", ticket.FileName),
			fmt.Sprintf("--checksum=sha-1=%s", sha1),
		)
		for name, value := range ticket.Headers {
			cmd.Args = append(cmd.Args, fmt.Sprintf("--header=%s: %s", name, value))
		}
		cmd.Args = append(cmd.Args, ticket.Url)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		return cmd.Run()
	} else {
		// TODO
		return nil
	}
}

func (c *DlCommand) curlDownload(conf *app.CurlConf, ticket *elevengo.DownloadTicket) error {
	cmd := exec.Command(conf.Path, "-#", ticket.Url)
	for name, value := range ticket.Headers {
		cmd.Args = append(cmd.Args, "-H", fmt.Sprintf("%s: %s", name, value))
	}
	cmd.Args = append(cmd.Args, "-o", ticket.FileName)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func (c *DlCommand) ImplCplt(ctx *context.Impl, index int, prefix string) (head string, choices []string) {
	head = ""
	if index == 0 {
		choices = ctx.Fs.FileNames(prefix)
	} else {
		choices = make([]string, 0)
	}
	return
}
