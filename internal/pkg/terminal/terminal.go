package terminal

import (
	"errors"
	"fmt"
	"github.com/peterh/liner"
	"strings"
)

var (
	errCommandNotExist = errors.New("no such command")
)

type Terminal struct {
	state *liner.State

	ctx  Context
	cmds map[string]Command
}

/*
Register one or more commands into terminal.
*/
func (t *Terminal) Register(cmds ...Command) {
	if t.cmds == nil {
		t.cmds = make(map[string]Command)
	}
	for _, cmd := range cmds {
		t.cmds[cmd.Name()] = cmd
	}
}

func (t *Terminal) Run() (err error) {
	if err = t.ctx.Startup(); err != nil {
		return err
	}
	for t.ctx.Alive() {
		if input, err := t.state.Prompt(t.ctx.Prompt()); err != nil {
			if err == liner.ErrPromptAborted {
				break
			} else {
				t.handleErr(err)
			}
		} else {
			t.handleErr(t.handle(input))
			t.state.AppendHistory(input)
		}
	}
	return t.ctx.Shutdown()
}

func (t *Terminal) handle(line string) (err error) {
	// Split input by space
	fields := split(line)
	if len(fields) == 0 {
		return
	}
	name, args := fields[0], fields[1:]
	if name == "" {
		return
	}
	if c, ok := t.cmds[name]; !ok {
		return errCommandNotExist
	} else {
		return c.Exec(t.ctx, args)
	}
}

func (t *Terminal) handleErr(err error) {
	if err == nil {
		return
	} else {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func (t *Terminal) wordCompleter(line string, pos int) (head string, choices []string, tail string) {
	// pre-init the result
	head, choices, tail = StringLeftRunes(line, pos), make([]string, 0), StringRightRunes(line, pos)
	// parse input
	fields := split(head)
	if len(fields) == 1 {
		// Here we need give choices for command names
		head, tail = "", ""
		for name, cmd := range t.cmds {
			if len(fields[0]) > 0 && !strings.HasPrefix(name, fields[0]) {
				continue
			}
			if cmd.HasArgs() {
				// Append a space to command which has arguments
				choices = append(choices, name+" ")
			} else {
				choices = append(choices, name)
			}
		}
	} else {
		// Here we need give choices for command's argument
		name := fields[0]
		cmd, ok := t.cmds[name]
		if !ok {
			return
		}
		ac, ok := cmd.(ArgCompleter)
		if !ok {
			return
		}
		// Now we find the command, and make sure it supports ArgCompleter.
		// Build the "head"
		fieldCount := len(fields)
		buf := strings.Builder{}
		buf.WriteString(name)
		buf.WriteString(" ")
		for i := 1; i < fieldCount-1; i++ {
			arg := fields[i]
			if len(arg) > 0 {
				buf.WriteString(arg)
				buf.WriteString(" ")
			}
		}
		head, tail = buf.String(), ""
		// Find last arg and call ArgCompleter.Completer
		index, lastArg := fieldCount-2, fields[fieldCount-1]
		var argHead string
		argHead, choices = ac.Completer(t.ctx, index, lastArg)
		if argHead != "" {
			head = fmt.Sprintf("%s%s", head, argHead)
		}
	}
	return
}

func New(ctx Context) *Terminal {
	// Create state
	state := liner.NewLiner()
	state.SetCtrlCAborts(true)
	state.SetTabCompletionStyle(liner.TabPrints)
	// Create terminal
	t := &Terminal{
		state: state,
		ctx:   ctx,
		cmds:  make(map[string]Command),
	}
	t.state.SetWordCompleter(t.wordCompleter)
	return t
}
