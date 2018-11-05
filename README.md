# NotGiphy

This is the backend service for [NotGiphyUI](https://github.com/corystone/notgiphyui).

NotGiphy is written in Go, with only [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) as a dependency.

## Quickstart instructions (apt)

### Get golang working
1. wget https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz
2. tar -C /usr/local -xzf go1.11.2.linux-amd64.tar.gz
3. export PATH=$PATH:/usr/local/go/bin

### Build NotGiphy
1. git clone https://github.com/corystone/notgiphy.git ~/go/src/github.com/corystone/notgiphy
2. apt-get install gcc libc6-dev make
3. cd ~/go/src/github.com/corystone/notgiphy
4. go get github.com/mattn/go-sqlite3
5. make

### Put the pieces together
1. Build [NotGiphyUI](https://github.com/corystone/notgiphyui) separately.
2. mkdir static
3. Copy all the files from notgiphyui/dist to static
4. If you have your own giphy api key: export NOTGIPHY\_API\_KEY=\<your\_api\_key\>
(Otherwise it will use a public beta key)
5. ./bin/notgiphy
