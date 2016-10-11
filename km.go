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
	var downloadlist []string
	links := GrabMainLinks("")
	cls()

	for index, k := range links {
		ver := strings.Replace(strings.Replace(k, mainline, "", 1), "/", "", 1)
		fmt.Printf("[%d] %s \n", index, ver)

	}
	fmt.Printf("\nChoose index: ")
	var i int
	fmt.Scan(&i)
	u := links[i]

	lst := GrabMainLinks(u)

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
	err := os.Mkdir(pth, 0777)
	pe(err)

	//rm dup
	for v := range downloadlist {
		if m[downloadlist[v]] != true {
			m[downloadlist[v]] = true
			uniq = append(uniq, downloadlist[v])
			// fmt.Println(downloadlist[v])
			// f := strings.Replace(downloadlist[v], mainline, "", 1)
			// Download(downloadlist[v], pth+"/"+f)
		}
	}

	var wg sync.WaitGroup

	for _, l := range uniq {
		wg.Add(1)
		f := strings.Split(strings.Replace(l, mainline, "", 1), "/")[1]
		go Download(l, pth+"/"+f, &wg)

	}
	wg.Wait()

	install(pth)

}

//Download download file
func Download(u, dest string, wg *sync.WaitGroup) {
	out, err := os.Create(dest)
	pe(err)
	fmt.Printf("file is downloading...\n")

	defer out.Close()

	response, err := http.Get(u)
	we(err)
	defer response.Body.Close()

	bs, err := io.Copy(out, response.Body)
	we(err)

	fmt.Println(bs, "file downloaded.")
	l(u, "Downloaded", dest)
	defer wg.Done()

}

func install(dest string) {
	files, err := ioutil.ReadDir(dest)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, v := range files {
		fname := v.Name()
		cmd := exec.Command("/usr/bin/sudo", "dpkg", "-i", fname) //dependency bug when installing header will be fixed
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
func main() {
	// DisplayMenu()
	DownloadKernel()
}
