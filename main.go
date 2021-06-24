package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleffmpeg"
	"layeh.com/gumble/gumbleutil"
	_ "layeh.com/gumble/opus"

	"github.com/avelino/slugify"
	"github.com/k3a/html2text"
)

type sourceFileTrimmed string

// SourceFileTrimmed is standard file source with trimmed silence
func SourceFileTrimmed(filename string) gumbleffmpeg.Source {
	return sourceFileTrimmed(filename)
}

func (s sourceFileTrimmed) Arguments() []string {
	return []string{"-i", string(s), "-af", "silenceremove=1:0:-30dB"}
}

func (sourceFileTrimmed) Start(*exec.Cmd) error {
	return nil
}

func (sourceFileTrimmed) Done() {
}

func main() {
	files := make(map[string]string)
	var stream *gumbleffmpeg.Stream

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [flags] [audio files...]\n", os.Args[0])
		flag.PrintDefaults()
	}

	var Reload = func() {
		for _, dir := range flag.Args() {
			filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				var file = info.Name()
				key := strings.Split(filepath.Base(file), ".")[0]
				if path == dir {
					return nil
				}
				files[key] = path
				return nil
			})
		}
	}
	var cucha *gumble.Channel

	gumbleutil.Main(gumbleutil.AutoBitrate, gumbleutil.Listener{
		Connect: func(e *gumble.ConnectEvent) {
			Reload()
			e.Client.Self.Register()
			root := e.Client.Channels[0]
			cucha = root.Find("la cucha de juancarlos")
		},

		TextMessage: func(e *gumble.TextMessageEvent) {
			if e.Sender == nil {
				return
			}

			msg := e.Message
			msg = html2text.HTML2Text(msg)

			fmt.Printf("%s: %s\n", e.Sender.Name, msg)

			// if msg == "x" {
			// msg = "!p xfiles"
			// }

			if len(msg) < 1 {
				return
			}

			if msg[:1] != "!" && (len(msg) > 1 && msg[1] != '!') {
				return
			}

			e.Client.Self.Move(e.Sender.Channel)

			if msg == "!" {
				stream.Stop()
				return
			}

			if msg == "!cucha" {
				stream.Stop()
				e.Client.Self.Move(cucha)
				return
			}

			if msg == "!random" {
				if stream != nil {
					stream.Stop()
				}
				keys := []string{}
				for k := range files {
					keys = append(keys, k)
				}

				rand.Seed(time.Now().Unix())

				name := keys[rand.Int()%len(keys)]
				msg = "!p " + name
				e.Sender.Channel.Send("!random = "+msg, false)
			}

			if msg == "!l" || msg == "l!" {
				Reload()

				s := []string{}
				for k, _ := range files {
					s = append(s, k)
				}
				sort.Strings(s)

				e.Sender.Channel.Send(strings.Join(s, ", "), false)
				return
			}

			if msg == "!t" || msg == "t!" {
				e.Sender.Channel.Send(stream.Elapsed().String(), false)
				return
			}

			if len(msg) < 3 {
				return
			}

			if msg[:3] == "!a " || msg[:3] == "a! " {
				parts := strings.SplitN(msg, " ", 3)
				if len(parts) < 3 {
					e.Sender.Channel.Send("? !a nombre https://...", false)
					return
				}

				name := slugify.Slugify(parts[1])
				link := parts[2]

				link2 := strings.Split(link, ">")
				if len(link2) > 1 {
					link3 := strings.Split(link2[1], "<")
					if len(link3) > 1 {
						link = link3[0]
					}
				}

				e.Sender.Channel.Send("ok...", false)
				cmd := exec.Command("/usr/bin/python3", "/home/user/.local/bin/youtube-dl",
					"-x", link,
					"-o", "audio/"+name+".%(id)s.$(ext)s")
				err := cmd.Run()
				if err != nil {
					e.Sender.Channel.Send(err.Error(), false)
				}
				Reload()
				e.Sender.Channel.Send("OK! !p "+name, false)

			}
			var key = ""
			if msg[:3] == "!p " || msg[:3] == "p! " {
				key = msg[3:]
			}
			if key == "" {
				return
			}
			file, ok := files[key]
			if !ok {
				e.Sender.Channel.Send("?", false)
				return
			}
			if stream != nil && stream.State() == gumbleffmpeg.StatePlaying {
				stream.Stop()
			}
			stream = gumbleffmpeg.New(e.Client, SourceFileTrimmed(file))
			if err := stream.Play(); err != nil {
				e.Sender.Channel.Send("err: "+err.Error(), false)
			} else {
				// e.Sender.Channel.Send("sale", false)
			}
		},
	})
}
