# RockSonic

CLI-tool to semi-automate downloading, converting and copying part of your music library hosted on a SubSonic compatible server.
**For now only synching favourites is implemented**

## How to install

Currently linux only. Windows and MacOS testers will be appreciated!
Just download the rocksonic binary from the sidebar/releases

## How to run

Check out the example folder for more details.
After compiling, run:

```bash
./main -url https://music.navidrome.example/rest -user jon -pass doe -mp3 -quality 3 -coversize 200
```

You can run --help to get up to date information on the cli parameters.

## Upcoming features

- Build system so you can just download it and don't have to compile it yourself (DONE)
- Select the playlist to synch (DONE)
- Choose between flat and nested folder structure (DONE)
- If switching between tree and flat, remove old structure (DONE)
- add \_mp3 to directory name if content is converted to mp3 (DONE)
