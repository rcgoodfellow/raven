package rvn

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// https://gist.github.com/elazarl/5507969
func CopyLocalFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

func ValidateURL(input string) *url.URL {
	parsedURL, err := url.Parse(input)
	if err != nil {
		log.Errorf("error validating URL: %v\n", err)
	}
	return parsedURL
}

// return a path, which we will create a directory tree with path[0]/path[1]/.../path[n]/image
func ParseURL(parsedURL *url.URL) (path string, image string, err error) {
	// Path is easier to use than RawPath
	remoteFullPath := parsedURL.Path
	splitPath := strings.Split(remoteFullPath, "/")
	// get the image name, dont let user specify qcow2
	// when rvn goes beyond qcow2, need to use correct format
	image = splitPath[len(splitPath)-1]
	// get the scheme used
	// create necessary variables
	var userName string
	var hostName string
	// now to create a directory tree from the path, omit scheme and opaque
	if parsedURL.Opaque != "" {
		err = errors.New("Opaque URL not implemented")
		return path, image, err
	}
	if parsedURL.User != nil {
		userName = parsedURL.User.Username()
		path = filepath.Join(userName, "/")
	}
	if parsedURL.Host != "" {
		hostName = parsedURL.Host
		path = filepath.Join(path, hostName, "/")
	}
	// ftp://user@host:/path will become user/host/path.../
	pathMinusImage := strings.Join(splitPath[:len(splitPath)-1], "/")
	path = filepath.Join(path, pathMinusImage, "/")
	return path, image, nil
}

func DownloadURL(parsedURL *url.URL, downloadPath string, imageName string) error {
	URIScheme := parsedURL.Scheme
	var err error
	// if no scheme for downloading file is provided, default to https
	// TODO: enforce HTTPS -- do not allow http, redirect
	if URIScheme == "https" {
		err = DownloadFile(filepath.Join(downloadPath, imageName), parsedURL.String())
	} else if URIScheme == "http" {
		err = errors.New("http is not supported, please use https!")
		return err
	} else if URIScheme == "" {
		DownloadFile(filepath.Join(downloadPath, imageName), parsedURL.String())
	} else {
		err := errors.New(parsedURL.Scheme + " is not currently implemented!")
		return err
	}
	return nil
}

// https://golangcode.com/download-a-file-from-a-url/
func DownloadFile(filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func CreateNetbootImage() error {
	netbootImage := "/var/rvn/img/netboot"
	_, err := os.Stat(netbootImage)
	if err != nil {
		cmd := exec.Command("qemu-img", "create", netbootImage, "25G")
		log.Printf("Creating netboot image")
		err := cmd.Run()
		return err
	}
	return nil
}
