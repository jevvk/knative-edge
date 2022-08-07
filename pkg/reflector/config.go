package reflector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func (r *Reflector) watchConfig(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		r.err <- fmt.Errorf("couldn't create fsnotify watcher: %s", err)
		return
	}

	if _, err := filepath.Abs(ConfigPath); err != nil {
		r.err <- fmt.Errorf("couldn't find config directory: %s", err)
		return
	}

	if err := watcher.Add(ConfigPath); err != nil {
		r.err <- fmt.Errorf("couldn't watch config directory: %s", err)
		return
	}

	tokenFile := fmt.Sprintf("%s/%s", ConfigPath, AuthenticationTokenFile)
	remoteURLFile := fmt.Sprintf("%s/%s", ConfigPath, RemoteURLFile)

	cfg := config{}

	if data, err := os.ReadFile(tokenFile); err == nil {
		cfg.token = string(data)
	}

	if data, err := os.ReadFile(remoteURLFile); err == nil {
		cfg.url = string(data)
	}

	r.reload <- &cfg

	for {
		select {
		case err := <-watcher.Errors:
			r.err <- err
			watcher.Close()
			return
		case ev := <-watcher.Events:
			if ev.Op&(fsnotify.Create|fsnotify.Write) <= 0 {
				continue
			}

			if ev.Name == tokenFile {
				data, err := os.ReadFile(tokenFile)

				if err != nil {
					continue
				}

				cfg.token = string(data)
			} else if ev.Name == remoteURLFile {
				data, err := os.ReadFile(remoteURLFile)

				if err != nil {
					continue
				}

				cfg.url = string(data)
			} else {
				continue
			}

			r.reload <- &cfg
		}
	}
}
