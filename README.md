# RockSonic

CLI-tool to semi-automate downloading, converting and copying part of your music library hosted on a SubSonic compatible server.
**For now only synching favourites is implemented**

## How to install

Currently you have to compile this project by installing golang on your machine and compiling it with `go build main.go`.
The automated built systems is currently broken. When it is fixed, you can just download the binaries and run them directly.

## How to run

Check out the example folder for more details.
After compiling, run:

```bash
./main -url https://music.navidrome.example/rest -user jon -pass doe -mp3 -quality 3 -coversize 200
```

You can run --help to get up to date information on the cli parameters.

## Upcoming features

- Build system so you can just download it and don't have to compile it yourself
- Select the playlist to synch (DONE)
- Choose between flat and nested folder structure (DONE)

## TODO

- remove removed songs from updated output folder
