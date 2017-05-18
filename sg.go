package main
import (
	"io/ioutil"
	"math"
	"bufio"
	"io"
	"path/filepath"
	"os"
	"fmt"
	"time"
	"sort"
	"strconv"
	"strings"
	"regexp"
)

const MAX_URL_LENGTH = 32
const POSTS_PER_LIST = 5


type List struct {
	posts []Post
}

type Post struct {
	path string
    title string
	unixtime int64
	date string
	content string
}

// CopyFile copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file. The file mode will be copied from the source and
// the copied data is synced/flushed to stable storage.
// https://gist.github.com/m4ng0squ4sh/92462b38df26839a3ca324697c8cba04
func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
// Symlinks are ignored and skipped.
// https://gist.github.com/m4ng0squ4sh/92462b38df26839a3ca324697c8cba04
func CopyDir(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if err == nil {
		return fmt.Errorf("destination already exists")
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}

func checkError(error_ error) {
	if error_ != nil {
		fmt.Printf("Error: %s", error_.Error())
		os.Exit(-1);
	}
}

func getFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = lines + "\n" + scanner.Text() //append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func createFile(path string, filename string, data string) {
	err := os.MkdirAll(path, 0666)
	checkError(err)
	
	f, err := os.Create(path + filename)
	checkError(err)
	defer f.Close()
	
	_, err = f.Write([]byte(data))
	checkError(err)
}

func generate() {
	var posts []Post //data structure for each post
	var lists []List //data structure for each post list
	
	//load post templates into memory
	postTemplate, err := getFile("_templates/post.template")
	checkError(err)
	
	listTemplate, err := getFile("_templates/list.template")
	checkError(err)
	
	linkTemplate, err := getFile("_templates/link.template")
	checkError(err)
	
	//get list of every file in _posts directory
	files, err := ioutil.ReadDir("_posts/")
	checkError(err)
	
	//compile the url-safety regexp
	rgx := regexp.MustCompile("[^a-zA-Z-0-9]")

	for _, f := range files { //for each file in the _posts/ directory...
        if( f.Name()[len(f.Name())-5:] == ".post") {//if the file extension is a .post...
			
			content, err := getFile("_posts/" + f.Name()); //load the content of the file
			checkError(err)
			
			t, err := time.Parse("2006-01-02", f.Name()[:10]) //generate a time object from the first 10 chars of the post filename
			checkError(err)
			
			date := t.Format("2006/01/02") //make the time pretty
			
			unixtime := t.Unix() //make the time functional
			
			folder := "posts/" + date + "/" //relative folder containing this post
				
			title := f.Name()[11:len(f.Name())-5] //cut off the date data and ".post", leaving just the title
			
			filename := strings.Replace(title, " ", "-", -1) //replace spaces with dashes because %20 sucks

			filename = string(rgx.ReplaceAll([]byte(filename), []byte(""))) //make it URL safe
			
			if(len(filename) > MAX_URL_LENGTH) { //truncate filename, if necessary
				filename = filename[:MAX_URL_LENGTH]
			}
			filename = filename + ".html"
			
			posts = append(posts, Post{path: folder + filename, title: title, unixtime: unixtime , date: date, content: content})
			
			//now generate a file from this data
			postData := postTemplate
			postData = strings.Replace(postData, "{{title}}", title, -1)
			postData = strings.Replace(postData, "{{date}}", date, -1)
			postData = strings.Replace(postData, "{{content}}", content, -1)
			
			createFile("_output/" + folder, filename, postData)
		}
    }
	
	//now sort them by timestamp
	sort.Slice(posts, func(i, j int) bool { return posts[i].unixtime < posts[j].unixtime })

	//generate lists
	for i, p := range posts { //for every post in the blog
		if (i % POSTS_PER_LIST == 0) { //if this is the first post of what should be a new list...
			lists = append(lists, List{}) //create a new list
		}
		pid := int(math.Floor(float64(i/POSTS_PER_LIST))) //which list the post will be in
		lists[pid].posts = append(lists[pid].posts, p) //insert the post data into this list
	}
	
	for i, list := range lists { //for each list
		listData := listTemplate //grab the list template
		listData = strings.Replace(listData, "{{title}}", "Page " + strconv.Itoa(i), -1) //insert the title of the list into this template
		
		linkData := ""
		for _, post := range list.posts { //for each post linked in this list...
			link := linkTemplate
			link = strings.Replace(link, "{{link}}", post.path, -1)
			link = strings.Replace(link, "{{title}}", post.title, -1)
			link = strings.Replace(link, "{{date}}", post.date, -1)
			linkData = linkData + link
		}
		listData = strings.Replace(listData, "{{content}}", linkData, -1) //insert these links into the list data
		
		createFile("_output/", strconv.Itoa(i) + ".html", listData) //and write.
	}
}

func main() {
	//delete everything from the _output directory
	err := os.RemoveAll("_output/")
	checkError(err)

	//copy all static files to the output folder
	err = CopyDir("_static/", "_output/")
	checkError(err)	
	
	generate();
}