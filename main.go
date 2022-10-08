package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/gen2brain/malgo"
	"github.com/go-audio/wav"
	"github.com/gorilla/websocket"
)

const SampleRate uint32 = 16000 // 8000 is too low
const OutFileName = "out.wav"

var (
	// a VOSK server address
	host string

	// set phrase to a limited dictionary to increase accuracy
	// not work for the model "vosk-model-en-us-0.22" (1.8G)
	// only works for the dynamic model "vosk-model-en-us-0.22-lgraph" (128M)
	limitWords bool
)

func init() {
	flag.StringVar(&host, "host", "127.0.0.1:2700", "")
	flag.BoolVar(&limitWords, "limitWords", false, "")
}

func main() {
	flag.Parse()

	u := url.URL{Scheme: "ws", Host: host, Path: ""}
	ws, _, err := websocket.DefaultDialer.DialContext(context.Background(), u.String(), nil)
	chk(err)

	defer func() {
		// send finial msg
		ws.WriteMessage(websocket.TextMessage, []byte(`{"eof" : 1}`))
		// read final msg
		ws.ReadMessage()
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		ws.Close()
	}()

	var bs string

	if !limitWords {
		bs = fmt.Sprintf(`
		{
			"config" : {
				"sample_rate" : %d,
				"words": 0
			}
		}`, SampleRate)
	} else {
		bs = fmt.Sprintf(`
		{
			"config" : {
				"sample_rate" : %d,
				"phrase_list" : [
					"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", 
					"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "zed",

					"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", 
					"eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen", 
					"twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety",
					"hundred", "thousand",
				],
				"words": 0
			}
		}`, SampleRate)
	}
	ws.WriteMessage(websocket.TextMessage, []byte(bs))

	go func() {
		for {
			_, msg, err2 := ws.ReadMessage()
			if err2 != nil {
				color.Red(err2.Error())
				break
			}

			if strings.Contains(string(msg), "text") {
				fmt.Println(string(msg))
			}
		}
	}()

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		fmt.Printf("LOG <%v>\n", message)
	})
	chk(err)

	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = SampleRate

	var playbackSampleCount uint32
	var capturedSampleCount uint32
	pCapturedSamples := make([]byte, 0)

	sizeInBytes := uint32(malgo.SampleSizeInBytes(deviceConfig.Capture.Format)) // == 2

	// ---- write to file ----
	wavFile, err := os.Create(OutFileName)
	if err != nil {
		panic(err)
	}
	enc := wav.NewEncoder(wavFile,
		int(SampleRate), // SampleRate
		16,              // BitDepth
		1,               // Channels
		1)               // 1 == PCM

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: func(_, pSample []byte, framecount uint32) {

			sampleCount := framecount * deviceConfig.Capture.Channels * sizeInBytes

			capturedSampleCount += sampleCount

			pCapturedSamples = append(pCapturedSamples, pSample...)

			// ws_.Write(pSample)
			ws.WriteMessage(websocket.BinaryMessage, pSample)

			single_frame_len := len(pSample) / int(framecount)

			for i := 0; i < int(framecount); i++ {
				enc.WriteFrame(pSample[i*single_frame_len : i*single_frame_len+single_frame_len])
			}
		},
	})
	chk(err)

	err = device.Start()
	chk(err)

	fmt.Println("Press Enter to stop recording...")
	fmt.Scanln()

	device.Stop()
	device.Uninit()

	enc.Close()
	wavFile.Close()
	color.Yellow("wav saved to file: %s", OutFileName)

	// ---- playback ----
	{
		deviceConfig = malgo.DefaultDeviceConfig(malgo.Playback)
		deviceConfig.Playback.Format = malgo.FormatS16
		deviceConfig.Playback.Channels = 1
		deviceConfig.SampleRate = SampleRate

		color.Blue("Playing...")

		device, err = malgo.InitDevice(ctx.Context, deviceConfig, malgo.DeviceCallbacks{
			Data: func(pSample, _ []byte, framecount uint32) {
				samplesToRead := framecount * deviceConfig.Playback.Channels * sizeInBytes
				if samplesToRead > capturedSampleCount-playbackSampleCount {
					samplesToRead = capturedSampleCount - playbackSampleCount
				}

				copy(pSample, pCapturedSamples[playbackSampleCount:playbackSampleCount+samplesToRead])

				playbackSampleCount += samplesToRead

				if playbackSampleCount == uint32(len(pCapturedSamples)) {
					playbackSampleCount = 0
				}
			},
		})
		chk(err)

		err = device.Start()
		chk(err)

		fmt.Println("Press Enter to quit...")
		fmt.Scanln()

		device.Stop()
		device.Uninit()
	}
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
