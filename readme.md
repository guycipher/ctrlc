## ctrlc - Clipboard logger
ctrlc logs every copy you make while running and makes them accessible via hosted realtime ui.

#### License 
..

#### Author 
Alex Padula

#### Versions
...........0.0.1

### Note
You must set your environment variable ```CTRLC_AES_32``` with a 32 byte secret key.

#### Building and running
Make sure you are in the ctrlc directory.
```
go build ctrlc.go -o ctrlc
```

Run
```
./ctrlc
```

Open browser and go to:
```
http://localhost:47222
```

#### Program Features
- Appends copied text into ctrlc.dat
- AES Encryption
- Ties to HTTP frontend with realtime socket communication. (Default port is 47222)
- ~~Basic authentication~~

#### Donate
You can donate Ether:

#### Required

##### Unix Ubuntu type systems
```
sudo apt install libx11-dev
```
```
sudo apt-cache policy libx11-dev
```
```
sudo apt install libxinerama-dev
```
```
sudo apt install xorg-dev
```

##### Windows type systems
https://www.glfw.org/download.html#:~:text=Windows%20pre%2Dcompiled%20binaries

