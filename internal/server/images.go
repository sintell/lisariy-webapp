package server

import (
	"errors"
	"os/exec"
	"path"
	"sync"

	"github.com/labstack/echo"
)

func newProcessedPicture(p *Picture) *ProcessedPicture {
	pp := &ProcessedPicture{Picture: p, wg: &sync.WaitGroup{}}
	pp.wg.Add(3)
	return pp
}

type ProcessedPicture struct {
	*Picture
	wg *sync.WaitGroup
}

type PicturesProcessor struct {
	hiResQueue chan *ProcessedPicture
	tnQueue    chan *ProcessedPicture
	pcQueue    chan *ProcessedPicture
	stop       chan struct{}
	errors     chan error
	logs       chan string
	stopped    bool
}

func NewPicturesProcessor(l echo.Logger) *PicturesProcessor {
	pp := &PicturesProcessor{
		errors: make(chan error),
		logs:   make(chan string),
	}
	go (func() {
		for {
			select {
			case err := <-pp.errors:
				l.Error(err)
			case log := <-pp.logs:
				l.Info(log)
			default:
				continue
			}
		}
	})()

	return pp
}

func (pp *PicturesProcessor) PutOriginal(p *Picture) chan interface{} {
	notify := make(chan interface{})
	go (func() {
		if !pp.stopped {
			ps := newProcessedPicture(p)
			pp.processTn(ps)
			pp.processPc(ps)
			pp.processHiRes(ps)
			ps.wg.Wait()
			notify <- nil
		}
		close(notify)
	})()
	return notify
}

func (pp *PicturesProcessor) processTn(p *ProcessedPicture) {
	pp.logs <- "[tn] processing " + p.OriginalSrc
	cmd := exec.Command(
		"vipsthumbnail", path.Join(imagesBasePath, p.OriginalSrc),
		"--size", "400>",
		"-o", path.Join(imagesBasePath, p.ThumbnailSrc.SrcX1)+"[strip]")
	data, err := cmd.CombinedOutput()
	if err != nil {
		pp.errors <- errors.New(string(data))
		return
	}
	pp.logs <- "[tn] finished " + p.OriginalSrc
	p.wg.Done()
}

func (pp *PicturesProcessor) processPc(p *ProcessedPicture) {
	pp.logs <- "[pc] processing " + p.OriginalSrc

	cmd := exec.Command(
		"vipsthumbnail", path.Join(imagesBasePath, p.OriginalSrc),
		"--size", "1024>",
		"-o", path.Join(imagesBasePath, p.ProcessedSrc.SrcX1)+"[strip]")
	data, err := cmd.CombinedOutput()
	if err != nil {
		pp.errors <- errors.New(string(data))
	}

	pp.logs <- "[pc] finished " + p.OriginalSrc
	p.wg.Done()
}

func (pp *PicturesProcessor) processHiRes(p *ProcessedPicture) {
	pp.logs <- "[hr] processing " + p.OriginalSrc

	cmd := exec.Command(
		"vipsthumbnail", path.Join(imagesBasePath, p.OriginalSrc),
		"--size", "800>",
		"-o", path.Join(imagesBasePath, p.ThumbnailSrc.SrcX2)+"[strip]")
	data, err := cmd.CombinedOutput()
	if err != nil {
		pp.errors <- errors.New(string(data))
	}

	cmd = exec.Command(
		"vipsthumbnail", path.Join(imagesBasePath, p.OriginalSrc),
		"--size", "2048>",
		"-o", path.Join(imagesBasePath, p.ProcessedSrc.SrcX2)+"[strip]")
	data, err = cmd.CombinedOutput()
	if err != nil {
		pp.errors <- errors.New(string(data))
	}
	pp.logs <- "[hr] finished " + p.OriginalSrc
	p.wg.Done()
}
