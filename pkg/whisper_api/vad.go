package whisper_api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/maxhawkins/go-webrtcvad"
)

// DURATION is a sweetener so that it is obvious what this control message is in response to
type DURATION int

// ControlMessage is a VAD type control message
type ControlMessage struct {
	Code DURATION
	data string
}

// The enumeratable types of DURATION
const (
	SILENCE DURATION = iota
	PAUSE
)

// The human readable output for a control message
var ops = map[DURATION]ControlMessage{
	SILENCE: ControlMessage{SILENCE, "Silence"},
	PAUSE:   ControlMessage{PAUSE, "Pause"},
}

// String will extract the human readable type for a control message
func (e DURATION) String() string {
	if op, found := ops[e]; found {
		return op.data
	}
	return "???"
}

// VadProcessor is responsible for configuring a VAD for a conversation
type VadProcessor struct {
	*webrtcvad.VAD
	ActiveAudio     *bytes.Buffer
	framesize       int
	sampleRateHertz int
	sample10msSize  int
	silenceTimeout  int //a silence is the smallest unit of nothing detected
	pauseTimeout    int //a pause is a unit of nothing detected, made up of silences
	currentSilence  int
	currentPause    int
}

func (v *VadProcessor) PreProcess(rawAudio chan []byte, someBytes []byte) {

	frame := make([]byte, v.framesize)
	r := bytes.NewBuffer(someBytes)

	for {
		n, err := io.ReadFull(r, frame)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			//fmt.Println("detected EOF", err)
			break
		}
		if err != nil {
			log.Fatal("error here ", err)
			break
		}

		if n > 0 {
			rawAudio <- frame
		} else {
			break
		}

	}
}

func NewVAD(sampleRateHertz, sample10msSize, mode int) (VadProcessor, error) {
	tmpVadProcessor, err := webrtcvad.New()
	if err != nil {
		return VadProcessor{}, err
	}
	var vProcessor VadProcessor
	vProcessor.VAD = tmpVadProcessor
	// 8kHz
	// XkHz => (X * 10) * 2
	//const sample10msSize = 160
	vProcessor.framesize = 160
	vProcessor.sample10msSize = sample10msSize
	vProcessor.sampleRateHertz = sampleRateHertz
	vProcessor.silenceTimeout = 500 //definition of a silence
	vProcessor.pauseTimeout = 3000  //length of silence before moving on
	vProcessor.currentSilence = 0   //length of current silence
	vProcessor.currentPause = 0     //ms until the next question
	vProcessor.ActiveAudio = new(bytes.Buffer)

	// set its aggressiveness mode, which is an integer between 0 and 3
	if err := vProcessor.VAD.SetMode(mode); err != nil { //agression cuts out background noise...
		return VadProcessor{}, err
	}

	if ok := vProcessor.VAD.ValidRateAndFrameLength(sampleRateHertz, sample10msSize); !ok {
		return VadProcessor{}, errors.New("invalid rate or frame length")
	}

	return vProcessor, nil
}

// Worker is responsible for the Voice Activity Detection, taking an input byte array and
//
//	returning an integer representing the duration of the silence.
func (v *VadProcessor) Worker(rawAudio chan []byte, vadControl chan DURATION) {

	fmt.Println("VAD working....")
	//var voiceSize int //seems to just track the total speaking time since last silence, leave for now.
	// run init to not get index not found
	v.restartSilence()

	for {
		select {
		//we need a context to cancel if necesssary
		//case <-manager.Canceled():
		//	return
		case frame := <-rawAudio:
			frameMs := len(frame)
			if frameMs != v.sample10msSize {
				log.Printf("vad: invalid frame size (%v) (%v) \n", frameMs, v.sample10msSize)
				continue
			}

			frameActive, err := v.VAD.Process(v.sampleRateHertz, frame)
			if err != nil {
				log.Printf("vad Process error: (%v) \n", err)
				continue
			}
			if frameActive {
				v.ActiveAudio.Write(frame)
			}
			v.saveResults(frameActive, frameMs)
			v.sendAndResetSilence(frameActive, vadControl)
		}
	}
}

func (v *VadProcessor) saveResults(frameActive bool, frameMs int) {

	// save size of silence and voice
	if frameActive {
		//restart silenceSize
		v.restartSilence()
	} else {
		// there is no voice in this sample
		v.saveSilenceResult(frameMs)
	}
}

func (v *VadProcessor) sendAndResetSilence(frameActive bool, vadControl chan DURATION) {

	//its just a check to see if the state has changed (i think) - not foolproof
	if frameActive {
		//vadControl <- 0
		return
	}
	if v.currentSilence == v.silenceTimeout*16 {
		v.currentSilence = 0
		vadControl <- SILENCE
	}
	if v.currentPause == v.pauseTimeout*16 {
		v.currentPause = 0
		vadControl <- PAUSE
	}
}

func (v *VadProcessor) saveSilenceResult(frameMs int) {

	v.currentSilence += frameMs
	v.currentPause += frameMs
}

func (v *VadProcessor) restartSilence() {

	v.currentSilence = 0
	v.currentPause = 0
}
