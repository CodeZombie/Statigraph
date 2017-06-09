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

//////////////////////////////
//todo:
//	add more comments.
//	stress test with 100 edge-case entries
//	ensure that webserver mode works properly.

const DEBUG_MODE = true //true if testing the site locally through your browser, false if running on webserver.
const MAX_URL_LENGTH = 32
const POSTS_PER_LIST = 5
const BLOG_SUBDIRECTORY = "blog"

var WEBSERV_ROOT = "file:///C:/Users/cazum/Documents/GitHub/Statigraph/_output/" //absolute directory to the output directory

type Post struct {
	filename string
	title    string
	unixtime int64
	date     string
	year     int
	month    int
	content  string
	path     string
}

func getWebRoot() string {
	if DEBUG_MODE == true {
		//dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		//checkError(err)
		return WEBSERV_ROOT
	}
	return "/"
}

func checkError(error_ error) {
	if error_ != nil {
		fmt.Printf("Error: %s", error_.Error())
		os.Exit(-1)
	}
}

func intInSlice(haystack_ []int, needle_ int) bool {
	for _, h := range haystack_ {
		if h == needle_ {
			return true
		}
	}
	return false
}

func getPosts(folder_ string) ([]Post, error) {
	var posts []Post //data structure for each post

	rgx := regexp.MustCompile("[^a-zA-Z-0-9]")

	files, err := ioutil.ReadDir(folder_) //get list of every file in _posts directory
	checkError(err)

	for _, f := range files { //for each file in the _posts/ directory...
		if f.Name()[len(f.Name())-5:] == ".post" { //if the file extension is a .post...
			content, err := readFile("_posts/" + f.Name()) //load the content of the file
			checkError(err)

			t, err := time.Parse("2006-01-02", f.Name()[:10]) //generate a time object from the first 10 chars of the post filename
			checkError(err)

			date := t.Format("2006/01/02") //make the time pretty

			unixtime := t.Unix() //make the time functional
			year := t.Year()
			month := int(t.Month())
			//folder := "posts/" + date + "/" //relative folder containing this post

			title := f.Name()[11 : len(f.Name())-5] //cut off the date data and ".post", leaving just the title

			filename := strings.Replace(title, " ", "-", -1) //replace spaces with dashes because %20 sucks

			filename = string(rgx.ReplaceAll([]byte(filename), []byte(""))) //make it URL safe

			if len(filename) > MAX_URL_LENGTH { //truncate filename, if necessary
				filename = filename[:MAX_URL_LENGTH]
			}
			filename = filename + ".html"

			path := BLOG_SUBDIRECTORY + "/" + date

			posts = append(posts, Post{filename: filename, path: path, year: year, month: month, title: title, unixtime: unixtime, date: date, content: content})
		}
	}

	return posts, nil
}

func savePosts(posts_ []Post) error {
	postTemplate, err := readFile("_templates/post.template")
	if err != nil {
		return err
	}

	for _, p := range posts_ {
		postData := postTemplate
		postData = strings.Replace(postData, "{{title}}", p.title, -1)
		postData = strings.Replace(postData, "{{date}}", p.date, -1)
		postData = strings.Replace(postData, "{{content}}", p.content, -1)
		postData = strings.Replace(postData, "{{rootdirectory}}", getWebRoot(), -1)
		err = createFile("_output/"+p.path+"/", p.filename, postData)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveList(postList_ [][]Post, directoryPath string) error {
	listTemplate, err := readFile("_templates/list.template")
	checkError(err)

	linkTemplate, err := readFile("_templates/link.template")
	checkError(err)

	for i, page := range postList_ { //for each list
		listpageData := listTemplate                                                           //grab the list template
		listpageData = strings.Replace(listpageData, "{{title}}", "Page "+strconv.Itoa(i), -1) //insert the title of the list into this template
		listpageData = strings.Replace(listpageData, "{{rootdirectory}}", getWebRoot(), -1)
		linkData := ""
		for _, post := range page { //for each post linked in this list...
			link := linkTemplate
			link = strings.Replace(link, "{{link}}", getWebRoot()+post.path+"/"+post.filename, -1)
			link = strings.Replace(link, "{{title}}", post.title, -1)
			link = strings.Replace(link, "{{date}}", post.date, -1)
			linkData = linkData + link
		}
		listpageData = strings.Replace(listpageData, "{{content}}", linkData, -1) //insert these links into the list data

		fname := strconv.Itoa(i)
		if fname == "0" {
			fname = "index"
		}

		err = createFile("_output/"+BLOG_SUBDIRECTORY+"/"+directoryPath, fname+".html", listpageData) //and write.
		checkError(err)
	}
	return nil
}

func createLists(posts_ []Post) error {
	var yearData []int
	//root lists are structured as [page][posts]
	var monthData = make(map[int][]int)
	//root lists are structured as [page][posts]

	var rootList [][]Post
	//yearlist is organized like [year][page][posts]
	var yearList = make(map[int][][]Post)
	var monthList = make(map[int]map[int][][]Post)

	//first, sort everything
	sort.Slice(posts_, func(i, j int) bool { return posts_[i].unixtime < posts_[j].unixtime })

	//figure out which years/months there are among all the posts
	for _, p := range posts_ {
		if monthData[p.year] == nil {
			monthData[p.year] = []int{}
			yearData = append(yearData, p.year)
		}
		if intInSlice(monthData[p.year], p.month) == false {
			monthData[p.year] = append(monthData[p.year], p.month)
		}
	}

	//genearte rootList
	for i, p := range posts_ { //for every post in the blog
		if i%POSTS_PER_LIST == 0 { //if this is the first post of what should be a new listpage...
			rootList = append(rootList, []Post{}) //create a new page
		}
		pid := int(math.Floor(float64(i / POSTS_PER_LIST))) //which page the post will be on...
		rootList[pid] = append(rootList[pid], p)            //insert the post data into this listpage
	}

	//generate yearlist
	for _, y := range yearData { //for every year
		i := 0
		for _, p := range posts_ { //for every post
			if p.year != y { //if this post is not from the year in the outer loop
				continue //skip
			}

			if i%POSTS_PER_LIST == 0 { //if this is the first post of what should be a new list page...
				yearList[y] = append(yearList[y], []Post{}) //create a new page
			}
			yearList[y][len(yearList[y])-1] = append(yearList[y][len(yearList[y])-1], p) //insert the post data into this listpage
			i += 1
		}
	}

	//generate monthlist

	for _, y := range yearData { //for every year
		monthList[y] = make(map[int][][]Post) //init that year's map
		for _, m := range monthData[y] {      //for every month in that year
			i := 0                     //init our iterator to 0
			for _, p := range posts_ { //for every post
				if p.year != y || p.month != m { //if this post is not within our month and year
					continue //skip it
				}
				if i%POSTS_PER_LIST == 0 { //if this is the first post of what should be a new list page...
					monthList[y][m] = append(monthList[y][m], []Post{}) //create a new page
				}
				monthList[y][m][len(monthList[y][m])-1] = append(monthList[y][m][len(monthList[y][m])-1], p) //insert the post data into this listpage
				i += 1
			}
		}
	}

	//save rootlist
	saveList(rootList, "")

	//save yearList
	for _, y := range yearData {
		saveList(yearList[y], strconv.Itoa(y)+"/")
		for _, m := range monthData[y] {
			//add padding...
			smonth := strconv.Itoa(m)
			if m < 10 {
				smonth = "0" + smonth
			}
			saveList(monthList[y][m], strconv.Itoa(y)+"/"+smonth+"/")
		}
	}
	return nil
}

func main() {

	//load every post file into a data structure
	posts, err := getPosts("_posts/")
	checkError(err)

	//delete everything from the _output directory
	err = os.RemoveAll("_output/")
	checkError(err)

	//copy all static files to the output folder
	err = CopyDir("_static/", "_output/")
	checkError(err)

	//create a file for every post
	err = savePosts(posts)
	checkError(err)

	//generate and save post lists
	err = createLists(posts)
	checkError(err)
}
