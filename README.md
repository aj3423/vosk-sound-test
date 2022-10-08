###  A Golang demo for how to use:
 - [VOSK](https://alphacephei.com/vosk/ "VOSK")  for speech recognition.
 - [miniaudio](https://github.com/mackron/miniaudio "miniaudio") for capturing audio input from **microphone**.

  
### What it does:
1. Capture audio input and send to a local voice recognition engine
2. Play back the captured sound
3. Record voice to a ".wav" file


### Usage:
1.  `docker run -d -p 2700:2700 alphacep/kaldi-en:latest`
	This runs official VOSK-server docker image.
2.  `./vosk-sound-test`
It starts capturing and displays the words you say, also saves audio to file **out.wav**
3. Press `<Enter>` to stop capturing and play back
4. Press `<Enter>` again to exit.

### Troubleshooting

1. Low sound quality

Maybe you're using some bluetooth airbuds like "airpods". For system like Linux, the input sound frequency is limited to 8000 at bluetooth stack, 16000 is a minimal frequency for VOSK to work well. A dedicated wired/wireless microphone should work.

2. It doesn't work at all

Open the system sound manager, verify the recording device while this program is capturing. Sometimes a wrong device is choosed by default.

### Build from source
0. Install [Golang](https://go.dev/dl/ "Golang") 
1. `git clone https://github.com/aj3423/vosk-sound-test`
2. `cd vosk-sound-test`
3. `go build .`
