package upload

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"videoUploadAndProcessing/pkg/acapela_api"
	"videoUploadAndProcessing/pkg/video_processing"
	"videoUploadAndProcessing/pkg/whisper_api"
)

type SegmentJob struct {
	SRTSegment    whisper_api.SRTSegment
	VideoPath     string
	Suffix        string
	SegmentIdx    int
	TempDirPrefix string
}

type SegmentWorker struct {
	ID          int
	JobQueue    chan SegmentJob
	SegmentPath *string
	SegmentIdx  int
}

func (w SegmentWorker) Start(wg *sync.WaitGroup, errors chan<- error) {
	go func() {
		for job := range w.JobQueue {
			log.Printf("SegmentWorker %d: Starting processing for segment %d", w.ID, job.SegmentIdx)
			// Convert text to speech
			audioSegment, err := acapela_api.ConvertTextToSpeechUsingAcapela(job.SRTSegment.Text, job.Suffix, job.SegmentIdx, job.TempDirPrefix)
			if err != nil {
				errors <- fmt.Errorf("SegmentWorker %d: failed to convert text to speech for segment %d: %v", w.ID, job.SegmentIdx, err)
				continue
			}

			log.Printf("SegmentWorker %d: Converted text to speech for segment %d", w.ID, job.SegmentIdx)
			// Merge the voice-over with the video segment and overwrite the original segment
			var mergedSegment string
			if strings.HasSuffix(job.VideoPath, ".mp4") {
				mergedSegment = strings.TrimSuffix(job.VideoPath, ".mp4") + "_merged.mp4"
			} else {
				mergedSegment = job.VideoPath + "_merged.mp4"
			}

			err = video_processing.MergeVideoAndAudioBySegments(job.VideoPath, audioSegment, mergedSegment, job.SegmentIdx)
			if err != nil {
				errors <- fmt.Errorf("SegmentWorker %d: failed to merge video and audio for segment %d: %v", w.ID, job.SegmentIdx, err)
				continue
			}

			err = video_processing.AddSubtitlesToSegment(mergedSegment, job.SRTSegment, mergedSegment, job.SegmentIdx)
			if err != nil {
				errors <- fmt.Errorf("SegmentWorker %d: failed to add subtitles to segment %d: %v", w.ID, job.SegmentIdx, err)
				continue
			}

			log.Printf("SegmentWorker %d: Merged video and audio for segment %d", w.ID, job.SegmentIdx)

			// Store the merged segment path at the location pointed to by SegmentPath
			*w.SegmentPath = mergedSegment
			// Add a log here to trace the stored path
			log.Printf("SegmentWorker %d: Stored merged segment path for segment %d: %s", w.ID, job.SegmentIdx, *w.SegmentPath)
		}

		// Add a log here to check the final value of *w.SegmentPath
		log.Printf("SegmentWorker %d: Final stored merged segment path: %s", w.ID, *w.SegmentPath)

		// Decrement the wait group counter when done
		wg.Done()
	}()
}

// Handles the logic for segment workers.
func ProcessSegmentJobs(voiceSegmentPaths []string, allSegmentPaths []string, srtSegments []whisper_api.SRTSegment, tempDirPrefix string) ([]string, error) {
	var wg sync.WaitGroup
	errors := make(chan error, len(voiceSegmentPaths))

	mergedSegments := make([]string, len(allSegmentPaths))
	copy(mergedSegments, allSegmentPaths)

	segmentWorkers := make([]SegmentWorker, len(voiceSegmentPaths))
	for i, voiceSegment := range voiceSegmentPaths {
		idx := indexOf(voiceSegment, allSegmentPaths)
		segmentWorkers[i] = SegmentWorker{
			ID:          i,
			JobQueue:    make(chan SegmentJob, 1),
			SegmentPath: &mergedSegments[idx],
			SegmentIdx:  idx,
		}
	}

	wg.Add(len(voiceSegmentPaths))

	for i := 0; i < len(segmentWorkers); i++ {
		segmentWorkers[i].Start(&wg, errors)
	}

	for i := 0; i < len(voiceSegmentPaths); i++ {
		segmentJob := SegmentJob{
			SRTSegment:    srtSegments[i],
			VideoPath:     voiceSegmentPaths[i],
			Suffix:        "Ryan22k_NT",
			SegmentIdx:    i,
			TempDirPrefix: tempDirPrefix, // 新增這行
		}
		segmentWorkers[i].JobQueue <- segmentJob
	}

	for i := 0; i < len(segmentWorkers); i++ {
		close(segmentWorkers[i].JobQueue)
	}

	wg.Wait()

	close(errors)
	for err := range errors {
		if err != nil {
			log.Printf("Error processing segment: %v", err)
			return nil, err
		}
	}
	return mergedSegments, nil
}

func indexOf(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1 // not found.
}
