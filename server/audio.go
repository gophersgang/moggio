package server

import (
	"fmt"
	"io"
	"time"

	"github.com/mjibson/mog/output"
)

func (srv *Server) audio() {
	var out output.Output
	var t chan interface{}
	var seek *Seek
	var dur time.Duration
	var err error
	send := func(v interface{}) {
		go func() {
			srv.ch <- v
		}()
	}
	tick := func() {
		const expected = 4096
		if seek == nil {
			return
		}
		next, err := seek.Read(expected)
		if len(next) > 0 {
			out.Push(next)
			send(cmdAdvanceTime(time.Duration(len(next)) * dur))
		}
		if err != nil {
			seek = nil
		}
		if err == io.ErrUnexpectedEOF {
			send(cmdRestartSong)
		} else if err != nil {
			send(cmdNext)
		}
	}
	doSeek := func(c cmdSeek) {
		if seek == nil {
			return
		}
		err := seek.Seek(time.Duration(c))
		if err != nil {
			send(cmdError(err))
		}
	}
	for {
		select {
		case <-t:
			tick()
		case c := <-srv.audioch:
			switch c := c.(type) {
			case audioStop:
				t = nil
			case audioPlay:
				t = make(chan interface{})
				close(t)
			case audioSetParams:
				out, err = output.Get(c.sr, c.ch)
				if err != nil {
					c.err <- fmt.Errorf("mog: could not open audio (%v, %v): %v", c.sr, c.ch, err)
					return
				}
				dur = time.Second / (time.Duration(c.sr * c.ch))
				seek = NewSeek(c.dur > 0, dur, c.play)
				t = make(chan interface{})
				close(t)
				c.err <- nil
			case cmdSeek:
				doSeek(c)
			default:
				panic("unknown type")
			}
		}
	}
}

type audioSetParams struct {
	sr   int
	ch   int
	dur  time.Duration
	play func(int) ([]float32, error)
	err  chan error
}

type audioStop struct{}

type audioPlay struct{}
