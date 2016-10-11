package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

//write err
func we(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

//panic err
func pe(err error) {
	if err != nil {
		panic(err)
	}
}

//logs
func l(v ...interface{}) {
	f, err := os.OpenFile("tmp.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	pe(err)
	defer f.Close()
	log.SetOutput(f)
	log.Println(v)
}

//cls clear screen
func cls() {
	cmd := exec.Command("clear")
	b, err := cmd.CombinedOutput()
	pe(err)
	fmt.Print(string(b))
}

//GetKernels returns list of linux-images
func GetKernels() (kernels []string) {
	var out bytes.Buffer
	cmd := exec.Command("ls", "/boot/")
	cmd.Stdout = &out
	err := cmd.Run()
	pe(err)
	lst := strings.Fields(out.String())
	for _, k := range lst {
		if strings.Index(k, "vmlinuz-") > -1 {
			kernels = append(kernels, k)
		}
	}
	return kernels
}

//GetHeaders returns list of header files
func GetHeaders() (headers []string) {
	var out bytes.Buffer
	cmd := exec.Command("ls", "/usr/src")
	cmd.Stdout = &out
	err := cmd.Run()
	pe(err)
	lst := strings.Fields(out.String())
	for _, k := range lst {
		if strings.Index(k, "linux-headers-") > -1 {
			headers = append(headers, k)
		}
	}
	return headers
}

//RemoveOldKernels removes old kernels
func RemoveOldKernels(removelist []string) {

	if len(removelist) > 0 {
		var cmd exec.Cmd
		for _, k := range removelist {
			fmt.Println(k, " is removing...")
			cmd = *exec.Command("sudo", "apt", "remove", "--purge", "-y", k)
			bs, err := cmd.Output()
			we(err)
			fmt.Println(string(bs))
			if cmd.ProcessState.Success() {

				fmt.Printf("Removed!\n\n")
				l(k, "Removed")
				// DisplayMenu()
			}
		}
	}
}

//FindBootedKernel booted kernel
func FindBootedKernel() string {
	cmd := exec.Command("uname", "-r")
	b, err := cmd.Output()
	pe(err)
	return strings.TrimSpace(string(b))
}

//DisplayMenu menu
func DisplayMenu() {
	cls()
	fmt.Printf("Installed kernels:\n\n")
	images := GetKernels()
	headers := GetHeaders()
	var removelist []string
	bk := FindBootedKernel()
	for index, im := range images {
		if strings.Index(im, bk) > -1 {
			fmt.Printf("[%d] %s *\n", index, strings.Replace(im, "vmlinuz-", "", 1))

		} else {
			fmt.Printf("[%d] %s\n", index, strings.Replace(im, "vmlinuz-", "", 1))

		}

	}
	fmt.Printf("\nChoose and index [0-%d] to remove or [-1] for exit: ", len(images)-1)
	var i int
	_, err := fmt.Scan(&i)
	if err != nil { // letter 0 fix
		DisplayMenu()
	}
	if i == -1 {
		os.Exit(0)
	}
	if i <= len(images) && i >= 0 {

		li := images[i]
		ver := strings.Split(strings.Replace(li, "vmlinuz-", "", 1), "-generic")[0]
		li = strings.Replace(li, "vmlinuz-", "linux-image-", 1)
		removelist = append(removelist, li)
		for _, h := range headers {

			if strings.Index(h, ver) > -1 {
				removelist = append(removelist, h)
			}
		}
		// fmt.Println(removelist)
		RemoveOldKernels(removelist)
	} else {
		DisplayMenu()
	}
}

func main() {

	DisplayMenu()

}
