package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"time"
)

func main() {
	urlFile := flag.String("file", "url_list.txt", "List of URLs to archive")
	batchLimit := flag.Int("limit", 50, "Archiving batch limit")
	timeout := flag.Int("timeout", 300, "Timeout per achive (in seconds)")
	binaryPath := flag.String("binpath", "./hbcrawler", "Path of the headless browser")

	flag.Parse()

	file, err := os.Open(*urlFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	var urlCnt = 0
	var batchCnt = 1
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 {
			continue
		}

		tmpURL, err := url.Parse(line)
		if err != nil {
			fmt.Printf("Error archiving %s. %v\n", line, err)
			continue
		}

		var rawURL string
		if tmpURL.Scheme == "" {
			rawURL = "http://" + line
		} else {
			rawURL = line
		}

		fmt.Printf("[%d::%d] Archiving %s\n", batchCnt, urlCnt, rawURL)
		ret := archiveURL(rawURL, *binaryPath, *timeout)
		for i := 1; ret != 0 && i <= 3; i++ {
			fmt.Println("Retry", i)
			ret = archiveURL(rawURL, *binaryPath, *timeout)
		}

		if urlCnt >= *batchLimit {
			r := bufio.NewReader(os.Stdin)
			fmt.Printf("Press any enter to continue. ")
			r.ReadString('\n')
			batchCnt++
			urlCnt = 0
		}
		urlCnt++
	}

}

func archiveURL(rawURL string, binaryPath string, timeout int) (ret int) {
	cmd := exec.Command(binaryPath,
		rawURL,
		"test",
		"--ssl-protocol=any",
		"--ignore-ssl-errors=yes",
		"--proxy=192.168.180.53:9090",
		"--user-agent=\"Mozilla/5.0 \\(X11\\; Linux x86_64\\) AppleWebKit/537.36 \\(KHTML, like Gecko\\) Chrome/32.0.1700.77 Safari/537.36\"")

	//fmt.Printf("%+v\n", cmd)
	err := cmd.Start()
	if err != nil {
		fmt.Println("Command not started.", err)
		return 1
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			fmt.Println("failed to kill process. ", err)
		}
		fmt.Println("process timeout-ed")
		ret = 2
		<-done
	case err := <-done:
		if err != nil {
			fmt.Println("process error.", err)
			ret = 2
		}
	}

	return ret
}
