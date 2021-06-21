package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/grafov/m3u8"
)

const playlist = "/Users/jameslucktaylor/Downloads/GB-Video-Grabber/E3-2021.m3u"

var actuallyDelete, verbose bool

func init() {
	flag.BoolVar(&actuallyDelete, "delete", false, "actually delete MP4 files not in the playlist")
	flag.BoolVar(&verbose, "verbose", false, "show more output")
}

func main() {
	flag.Parse()

	playlistDir, err := ioutil.ReadDir(path.Dir(playlist))
	if err != nil {
		fmt.Printf("error reading directory: %v\n", err)

		return
	}

	files := map[string]struct{}{}

	for _, osfi := range playlistDir {
		if osfi.IsDir() {
			continue
		}

		files[osfi.Name()] = struct{}{}
	}

	fmt.Printf("Found %d files in playlist directory '%s'.\n", len(files), path.Dir(playlist))

	fmt.Print("Parsing playlist... ")

	startParse := time.Now().UnixNano()

	f, err := os.Open(playlist)
	if err != nil {
		fmt.Printf("error opening playlist: %v\n", err)

		return
	}

	p, listType, err := m3u8.DecodeFrom(bufio.NewReader(f), true)
	if err != nil {
		fmt.Printf("error decoding playlist: %v\n", err)

		return
	}

	if listType != m3u8.MEDIA {
		fmt.Printf("unknown list type '%v'\n", listType)

		return
	}

	mediapl := p.(*m3u8.MediaPlaylist) //nolint:errcheck // Covered by listType return

	filesToKeep := map[string]struct{}{}

	for _, seg := range mediapl.Segments {
		if seg == nil {
			continue
		}

		decoded, errPQ := url.ParseQuery(seg.URI)
		if errPQ != nil {
			fmt.Printf("error parsing URI: '%v'\n", listType)

			return
		}

		for key := range decoded {
			fileToKeep := path.Join(path.Dir(playlist), key)
			filesToKeep[fileToKeep] = struct{}{}
		}
	}

	finishParse := time.Now().UnixNano()
	parseSeconds := float64((finishParse - startParse)) / 1e9

	fmt.Printf("done in %s.\n", humanize.SI(parseSeconds, "s"))
	fmt.Printf("Keeping %d items from playlist.\n", len(filesToKeep))

	savedSpace := uint64(0)

	for file := range files {
		fileInDir := path.Join(path.Dir(playlist), file)

		if !strings.EqualFold(path.Ext(fileInDir), ".mp4") {
			fmt.Printf("'%s' is not an MP4 file.\n", fileInDir)

			continue
		}

		if _, exists := filesToKeep[fileInDir]; exists {
			if verbose {
				fmt.Printf("Keeping '%s'.\n", fileInDir)
			}

			continue
		}

		stat, err := os.Stat(fileInDir)
		if err != nil {
			fmt.Printf("could not stat file '%s': %v\n", fileInDir, err)

			return
		}

		savedSpace += uint64(stat.Size())

		if !actuallyDelete {
			fmt.Printf("[DRY RUN] Would delete '%s'.\n", fileInDir)

			continue
		}

		fmt.Printf("Deleting '%s'!\n", fileInDir)

		err = os.Remove(fileInDir)
		if err != nil {
			fmt.Printf("error deleting file '%s': %v\n", fileInDir, err)

			return
		}
	}

	if actuallyDelete {
		fmt.Printf("Saved %s of space.\n", humanize.Bytes(savedSpace))
	} else {
		fmt.Printf("Would save %s of space.\n", humanize.Bytes(savedSpace))
	}
}
