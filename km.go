package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const (
	mainline     = "http://kernel.ubuntu.com/~kernel-ppa/mainline/"
	downloadPath = "/tmp/kernelz"
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
		if strings.Index(k, "linux-headers-") > -1 && !strings.Contains(k, "generic") {
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

//RemoveKernelMenu menu
func RemoveKernelMenu() {
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
	fmt.Printf("\nChoose an index [0-%d] to remove or [-1] for exit: ", len(images)-1)
	var i int
	_, err := fmt.Scan(&i)
	if err != nil { // letter 0 fix
		RemoveKernelMenu()
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
		RemoveOldKernels(removelist)
	} else {
		RemoveKernelMenu()
	}
}

//GrabMainLinks parse html and grab href
func GrabMainLinks(u string) (links []string) {
	if u == "" {
		u = mainline
	}
	resp, err := http.Get(u)
	defer resp.Body.Close()
	pe(err)

	tk := html.NewTokenizer(resp.Body)

	for t := tk.Next(); t != html.ErrorToken; {

		if t == html.StartTagToken {
			for _, a := range tk.Token().Attr {
				if a.Key == "href" {
					if u == mainline {
						if strings.Index(a.Val, "v4") > -1 { // only v4+
							links = append(links, mainline+a.Val)
						}
					} else {
						links = append(links, u+a.Val)

					}

				}
			}
		}
		t = tk.Next()
	}

	return links
}

//DownloadKernel downlods deb files
func DownloadKernel() {

	var wg sync.WaitGroup
	var downloadlist []string
	links := GrabMainLinks("")
	cls()
	if len(links) < 1 {
		os.Exit(1)
	}
	for index, k := range links {
		ver := strings.Replace(strings.Replace(k, mainline, "", 1), "/", "", 1)
		fmt.Printf("[%d] %s \n", index, ver)

	}
	fmt.Printf("\nChoose an index to install or [-1] for exit: ")
	var i int
	_, err := fmt.Scan(&i)
	if err != nil {
		DownloadKernel()
	}
	if i == -1 {
		os.Exit(0)
	}
	u := links[i]

	lst := GrabMainLinks(u)
	if len(lst) < 1 {
		os.Exit(1)
	}
	m := map[string]bool{}
	uniq := []string{}
	for _, l := range lst {
		if strings.Index(l, "_all") > -1 {
			downloadlist = append(downloadlist, l)
		}
		if strings.Index(l, runtime.GOARCH+".deb") > -1 {
			if strings.Index(l, "generic") > -1 {
				downloadlist = append(downloadlist, l)

			}
		}
	}
	pth := downloadPath + fmt.Sprintf("%d", time.Now().Unix())
	err = os.Mkdir(pth, 0755)
	pe(err)

	//rm dup
	for v := range downloadlist {
		if m[downloadlist[v]] != true {
			m[downloadlist[v]] = true
			uniq = append(uniq, downloadlist[v])
		}
	}

	for _, l := range uniq {
		wg.Add(1)
		f := strings.Split(strings.Replace(l, mainline, "", 1), "/")[1]
		go Download(l, pth+"/"+f, &wg)

	}
	fmt.Printf("files downloading...\n")
	wg.Wait()

	install(pth)

}

//Download download file
func Download(u, dest string, wg *sync.WaitGroup) {
	out, err := os.Create(dest)
	pe(err)
	fname := strings.Split(strings.Replace(u, mainline, "", -1), "/")[1]

	defer out.Close()

	response, err := http.Get(u)
	we(err)
	defer response.Body.Close()

	bs, err := io.Copy(out, response.Body)
	we(err)

	fmt.Println(fname, bs, "bytes downloaded.")
	l(u, "Downloaded", dest)
	defer wg.Done()

}

//install downloaded deb files
func install(dest string) {
	files, err := ioutil.ReadDir(dest)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, v := range files {
		fname := v.Name()
		cmd := exec.Command("/usr/bin/sudo", "dpkg", "-i", fname) //dependency exit status 1
		cmd.Dir = dest
		by, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(err)
		}

		if cmd.ProcessState.Success() {
			fmt.Println(string(by))
			l("installed", v.Name())
		}

	}
}

//basic ui
func app() {
	cls()
	fmt.Printf("simple kernel management tool for ubuntu\n\n")
	fmt.Printf("[r] Remove old kernel\n[i] Install new kernel v4+\n[q] quit\n:? ")
	var s string
	fmt.Scan(&s)
	switch s {
	case "r":
		RemoveKernelMenu()
	case "i":
		DownloadKernel()
	case "q":
		os.Exit(0)
	default:
		app()
	}
}
func main() {
	app()
}
