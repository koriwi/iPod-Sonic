#!/bin/bash

# download your liked songs, convert to mp3 with quality 3 [1 is highest, 10 is lowest], and embedd covers with a width of 200px

./rocksonic -url https://music.navidrome.example/rest -user jon -pass doe -mp3 -quality 3 -coversize 400
