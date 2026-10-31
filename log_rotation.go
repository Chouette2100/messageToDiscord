package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/Chouette2100/exsrapi/v2"
	"github.com/Chouette2100/srapi/v2"
)

const logRotateAtEnv = "DISCORD_LOG_ROTATE_AT"

func setupDailyLogRotation(version string) (func(), error) {
	return setupDailyLogRotationAt(version, os.Getenv(logRotateAtEnv))
}

func setupDailyLogRotationAt(version, rotateAt string) (func(), error) {
	hour, minute, err := parseRotateAt(rotateAt)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", logRotateAtEnv, err)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	isForeground := term.IsTerminal(int(os.Stdout.Fd()))
	logfile, err := exsrapi.CreateLogfile(version, exsrapi.Version, srapi.Version)
	if err != nil {
		return nil, err
	}

	setOutput := func(file *os.File) {
		if isForeground {
			log.SetOutput(io.MultiWriter(file, os.Stdout))
			return
		}
		log.SetOutput(file)
	}
	setOutput(logfile)

	stop := make(chan struct{})
	done := make(chan struct{})
	var once sync.Once

	go func() {
		defer close(done)
		current := logfile

		for {
			next := nextRotationTime(time.Now(), hour, minute)
			timer := time.NewTimer(time.Until(next))

			select {
			case <-timer.C:
				if !shouldRotateNow(time.Now(), next) {
					continue
				}
				newLogfile, err := exsrapi.CreateLogfile(version, exsrapi.Version, srapi.Version)
				if err != nil {
					log.Printf("failed to rotate log file: %v", err)
					continue
				}
				setOutput(newLogfile)
				_ = current.Close()
				current = newLogfile
				log.Printf("log rotated at %02d:%02d", hour, minute)
			case <-stop:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				_ = current.Close()
				return
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(stop)
			<-done
		})
	}, nil
}

func parseRotateAt(rotateAt string) (int, int, error) {
	trimmed := strings.TrimSpace(rotateAt)
	if trimmed == "" {
		return 0, 0, nil
	}

	parts := strings.Split(trimmed, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("must be HH:MM")
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("hour must be 00-23")
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("minute must be 00-59")
	}

	return hour, minute, nil
}

func nextRotationTime(now time.Time, hour, minute int) time.Time {
	year, month, day := now.Date()
	loc := now.Location()
	next := time.Date(year, month, day, hour, minute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func shouldRotateNow(now, scheduled time.Time) bool {
	return !now.Before(scheduled)
}
