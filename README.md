# NotGiphy

This is the backend service for [NotGiphyUI](https://github.com/ston9665/notgiphyui).

NotGiphy is written in Go, with only [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) as a dependency.

## Quickstart instructions (apt)

### Get golang working
1. wget https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz
2. tar -C /usr/local -xzf go1.11.2.linux-amd64.tar.gz
3. export PATH=$PATH:/usr/local/go/bin

### Install NotGiphy
1. git clone https://github.com/ston9665/notgiphy.git ~/go/src/github.com/ston9665/notgiphy
2. apt-get install gcc libc6-dev make
3. cd ~/go/src/github.com/ston9665/notgiphy
4. go get github.com/mattn/go-sqlite3
5. make

### Make it work
1. build [NotGiphyUI](https://github.com/ston9665/notgiphyui)
2. mkdir static
3. copy all the files from notgiphyui/dist to static
4. if you have your own giphy api key: export NOTGIPHY\_API\_KEY=\<your\_api\_key\>
5. ./bin/notgiphy
