package kv

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
)

// Small helper to check if a given bit is set in a bitset -- fsnotify.Op
// is effectively a bit-packed integer where each bit is a flag (such as fsnotify.Write)
func isBitSet(bitset, bit fsnotify.Op) bool {
	return bitset&bit != 0
}

/*
 * FileShardMap reads a file from disk in JSON format and parses it as a ShardMap.
 * This file is continually watched for updates, propagating any updates to the
 * file to the ShardMap itself via calls to Update().
 *
 * This ShardMap file will typically be shared between nodes and clients. In a production
 * setting it may be distributed by your configuration management or cluster management
 * software (such as a ConfigMap in Kubernetes).
 *
 * For clean shutdown, call Shutdown() to stop the file-watching process.
 */
type FileShardMap struct {
	ShardMap ShardMap
	filename string
	watcher  *fsnotify.Watcher

	done chan struct{}
}

/*
 * Read a ShardMap from the given filename and watch it for any future updates.
 * May fail and return (nil, error) if the file does not exist, cannot be watched,
 * or cannot be parsed from JSON.
 *
 * After the initial load, any errors in reading the file or parsing it may be logged
 * but otherwise ignored and the last known good value will be used.
 */
func WatchShardMapFile(filename string) (*FileShardMap, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Errorf("could created a file watcher for %s: %q", filename, err)
		return nil, err
	}
	err = watcher.Add(filepath.Dir(filename))
	if err != nil {
		logrus.Errorf("could not watch file %s: %q", filename, err)
		return nil, err
	}

	fileSm := &FileShardMap{
		filename: filename,
		watcher:  watcher,
	}
	err = fileSm.applyFromFile()
	if err != nil {
		return nil, err
	}
	go fileSm.watchForUpdates()
	return fileSm, nil
}

func (fileSm *FileShardMap) Shutdown() {
	fileSm.done <- struct{}{}
}

func (fileSm *FileShardMap) watchForUpdates() {
	for {
		select {
		case event, ok := <-fileSm.watcher.Events:
			if !ok {
				logrus.Debugf("done watching shardmap file: %s", fileSm.filename)
				break
			}
			if event.Name != fileSm.filename {
				logrus.Tracef("ignoring update event for file: %s", event.Name)
				continue
			}

			if isBitSet(event.Op, fsnotify.Write) || isBitSet(event.Op, fsnotify.Create) {
				logrus.Debugf("shard map updated from %s (%s)", event.Name, event.Op.String())
				err := fileSm.applyFromFile()
				if err != nil {
					logrus.Warnf("failed to apply shardmap update -- using old shardmap value")
				}
			}
		case err, ok := <-fileSm.watcher.Errors:
			if !ok {
				logrus.Errorf("done after error watching shardmap file: %s", fileSm.filename)
				break
			}
			logrus.Warnf("error while watching shardmap directory %s: %q", fileSm.filename, err)
		case <-fileSm.done:
			return
		}
	}
}

func (fileSm *FileShardMap) applyFromFile() error {
	data, err := os.ReadFile(fileSm.filename)
	if err != nil {
		logrus.Errorf("failed to read shardmap file %s: %q", fileSm.filename, err)
		return err
	}

	var smState ShardMapState
	err = json.Unmarshal(data, &smState)
	if err != nil {
		logrus.Errorf("failed to unmarshall shardmap file %s: %q", fileSm.filename, err)
		return err
	}

	fileSm.ShardMap.Update(&smState)
	return nil
}
