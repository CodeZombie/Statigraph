package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const MAX_URL_LENGTH = 32
const POSTS_PER_LIST = 5
const BLOG_SUBDIRECTORY = "blog"

type Listpage struct {
	posts []Post
}

type Post struct {
	filename string
	title    string
	unixtime int64
	date     string
	content  string
	paths    []string
}

func checkError(error_ error) {
	if error_ != nil {
		fmt.Printf("Error: %s", error_.Error())
		os.Exit(-1)
	}
}

func getPosts(folder_ string) ([]Post, error) {
	var posts []Post //data structure for each post
	rgx := regexp.MustCompile("[^a-zA-Z-0-9]")

	files, err := ioutil.ReadDir(folder_) //get list of every file in _posts directory
	if err != nil {
		return nil, err
	}

	for _, f := range files { //for each file in the _posts/ directory...
		if f.Name()[len(f.Name())-5:] == ".post" { //if the file extension is a .post...

			content, err := readFile("_posts/" + f.Name()) //load the content of the file
			checkError(err)

			t, err := time.Parse("2006-01-02", f.Name()[:10]) //generate a time object from the first 10 chars of the post filename
			checkError(err)

			date := t.Format("2006/01/02") //make the time pretty

			unixtime := t.Unix() //make the time functional

			//folder := "posts/" + date + "/" //relative folder containing this post

			title := f.Name()[11 : len(f.Name())-5] //cut off the date data and ".post", leaving just the title

			filename := strings.Replace(title, " ", "-", -1) //replace spaces with dashes because %20 sucks

			filename = string(rgx.ReplaceAll([]byte(filename), []byte(""))) //make it URL safe

			if len(filename) > MAX_URL_LENGTH { //truncate filename, if necessary
				filename = filename[:MAX_URL_LENGTH]
			}
			filename = filename + ".html"

			var paths []string

			paths = append(paths, t.Format("02/"))
			paths = append(paths, t.Format("01/02/"))
			paths = append(paths, t.Format("2006/01/02/"))

			posts = append(posts, Post{filename: filename, paths: paths, title: title, unixtime: unixtime, date: date, content: content})
		}
	}

	sort.Slice(posts, func(i, j int) bool { return posts[i].unixtime < posts[j].unixtime })

	return posts, nil
}

func getList(posts_ []Post) ([]Listpage, error) {
	var list []Listpage //an array of lists

	for i, p := range posts_ { //for every post in the blog
		if i%POSTS_PER_LIST == 0 { //if this is the first post of what should be a new listpage...
			list = append(list, Listpage{}) //create a new listpage
		}
		pid := int(math.Floor(float64(i / POSTS_PER_LIST))) //which listpage the post will be in
		list[pid].posts = append(list[pid].posts, p)        //insert the post data into this listpage
	}

	return list, nil
}

func createPosts(posts_ []Post) error {
	postTemplate, err := readFile("_templates/post.template")
	if err != nil {
		return err
	}

	for _, p := range posts_ {
		postData := postTemplate
		postData = strings.Replace(postData, "{{title}}", p.title, -1)
		postData = strings.Replace(postData, "{{date}}", p.date, -1)
		postData = strings.Replace(postData, "{{content}}", p.content, -1)

		err = createFile("_output/"+BLOG_SUBDIRECTORY+"/"+p.paths[2], p.filename, postData)
		if err != nil {
			return err
		}
	}
	return nil
}

func createList(list_ []Listpage, directoryDepth_ int) error {
	listTemplate, err := readFile("_templates/list.template")
	if err != nil {
		return err
	}

	linkTemplate, err := readFile("_templates/link.template")
	if err != nil {
		return err
	}
	//the number of times "../" needs to appear before static resources from the list
	reversePath := ""
	for n := 0; n < (directoryDepth_ - 2); n++ {
		reversePath = reversePath + "../"
	}

	for i, listpage := range list_ { //for each list
		listpageData := listTemplate                                                           //grab the list template
		listpageData = strings.Replace(listpageData, "{{title}}", "Page "+strconv.Itoa(i), -1) //insert the title of the list into this template
		listpageData = strings.Replace(listpageData, "{{reversepath}}", reversePath, -1)
		linkData := ""
		for _, post := range listpage.posts { //for each post linked in this list...
			link := linkTemplate
			link = strings.Replace(link, "{{link}}", post.paths[directoryDepth_]+post.filename, -1)
			link = strings.Replace(link, "{{title}}", post.title, -1)
			link = strings.Replace(link, "{{date}}", post.date, -1)
			linkData = linkData + link
		}
		listpageData = strings.Replace(listpageData, "{{content}}", linkData, -1) //insert these links into the list data

		err = createFile("_output/"+BLOG_SUBDIRECTORY+"/", strconv.Itoa(i)+".html", listpageData) //and write.
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	//load every post file into a data structure
	posts, err := getPosts("_posts/")
	checkError(err)

	//generate a list, organizing every post by date
	defaultList, err := getList(posts)
	checkError(err)

	//delete everything from the _output directory
	err = os.RemoveAll("_output/")
	checkError(err)

	//copy all static files to the output folder
	err = CopyDir("_static/", "_output/")
	checkError(err)

	//create a file for every post
	err = createPosts(posts)
	checkError(err)

	//create a file for every list
	err = createList(defaultList, 2)
	checkError(err)
}
