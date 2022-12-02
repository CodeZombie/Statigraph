package main
import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/quick"
)


type Post struct {
	title    string
	unixtime int64
	date     string
	year     int
	month    int
	content  string
	path     string
}

func fail_on_error(error_ error) {
	/* Prints an error and kills the application if the error is not nil */
	if error_ != nil {
		fmt.Printf("Error: %s", error_.Error())
		os.Exit(-1)
	}
}

func get_posts(input_post_directory string, blog_subdirectory string, max_url_length int) ([]Post, error) {
	/* Reads every file in the input_post_directory into a slice of Post structs and returns. */
	var posts []Post

	rgx := regexp.MustCompile("[^a-zA-Z-0-9]")

	files, err := ioutil.ReadDir(input_post_directory) //get list of every file in INPUT_POST_DIRECTORY directory
	fail_on_error(err)

	for _, f := range files { //for each file in the INPUT_POST_DIRECTORY/ directory...
		if f.Name()[len(f.Name())-5:] == ".post" { //if the file extension is .post...

			content, err := read_file(filepath.Join(input_post_directory, f.Name())) //load the content of the file
			fail_on_error(err)

			document, err := goquery.NewDocumentFromReader(strings.NewReader(content))
			fail_on_error(err)

			//find every sg_codeblock
			//find it's 'language' attribute's value
			//replace the content of this tag with the chroma.Quick output
			document.Find("sg_codeblock").Each(func(i int, s *goquery.Selection) {
				
				//Find the "language" attr in the tag.
				language, attributeExists := s.Attr("language")
				if !attributeExists{
					fail_on_error(errors.New("<sg_codeblock> #%d does not have a \"language\" attribute."))
					return
				}

				//Get the contents of the <sg_codeblock> element
				innerText := s.Text()
				
				//Use Chroma to syntax-highlight the contents of the element
				var formattedStringBuilder strings.Builder
				formatters.Register("badnoise_style", html.New(html.Standalone(false), html.WithClasses(true)))
				err = quick.Highlight(&formattedStringBuilder, innerText, language, "badnoise_style", "abap")
				fail_on_error(err)
				
				//Set the html of the element to the chroma syntax highlighted version.
				formatted_text := formattedStringBuilder.String()
				s.SetHtml(formatted_text)
			})
			
			formatted_html, err := document.Html()
			fail_on_error(err)

			var input_date_format = "2006-01-02"
			post_time, err := time.Parse(input_date_format, f.Name()[ : len(input_date_format)]) //generate a time object from the first 10 chars of the post filename
			fail_on_error(err)

			title := f.Name()[len(input_date_format) : len(f.Name())-5] //cut off the date data and ".post", leaving just the title

			foldername := strings.Replace(title, " ", "-", -1) //replace spaces with dashes because `%20`` is ugly

			foldername = string(rgx.ReplaceAll([]byte(foldername), []byte(""))) //make it URL safe

			if len(foldername) > max_url_length { //truncate foldername, if necessary
				foldername = foldername[:max_url_length]
			}
			
			posts = append(posts, Post{
				path: filepath.Join(blog_subdirectory, foldername), 
				year: post_time.Year(), 
				month: int(post_time.Month()), 
				title: title, 
				unixtime: post_time.Unix(), 
				date: post_time.Format("Jan 02, 2006"), 
				content: formatted_html})
		}
	}

	return posts, nil
}

//TODO: replace this with the built-in html/template module
func save_posts(posts []Post, output_directory string, input_template_directory string) error {
	
	postTemplate, err := read_file(filepath.Join(input_template_directory, "post.template"))
	fail_on_error(err)

	for _, p := range posts {
		postData := postTemplate
		postData = strings.Replace(postData, "{{title}}", p.title, -1)
		postData = strings.Replace(postData, "{{date}}", p.date, -1)
		postData = strings.Replace(postData, "{{content}}", p.content, -1)
		err = create_file(filepath.Join(output_directory, p.path), "index.html", postData)
		if err != nil {
			return err
		}
	}
	return nil
}

func createLists(posts_ []Post, input_template_directory string, output_directory string, blog_subdirectory string, posts_per_list int) error {
	var rootList [][]Post

	//Sort posts by their post-date
	sort.Slice(posts_, func(i, j int) bool { return posts_[i].unixtime < posts_[j].unixtime })

	//genearte rootList
	for i, p := range posts_ { //for every post in the blog
		if i % posts_per_list == 0 { //if this is the first post of what should be a new listpage...
			rootList = append(rootList, []Post{}) //create a new page
		}
		pid := int(math.Floor(float64(i / posts_per_list))) //which page the post will be on...
		rootList[pid] = append(rootList[pid], p)            //insert the post data into this listpage
	}

	listTemplate, err := read_file(filepath.Join(input_template_directory, "list.template"))
	fail_on_error(err)

	linkTemplate, err := read_file(filepath.Join(input_template_directory, "link.template"))
	fail_on_error(err)

	for i, page := range rootList { //for each list
		listpageData := listTemplate                                                           //grab the list template
		listpageData = strings.Replace(listpageData, "{{title}}", "Page " + strconv.Itoa(i), -1) //insert the title of the list into this template
		linkData := ""
		for _, post := range page { //for each post linked in this list...
			link := linkTemplate
			link = strings.Replace(link, "{{link}}", "../" + post.path, -1)
			link = strings.Replace(link, "{{title}}", post.title, -1)
			link = strings.Replace(link, "{{date}}", post.date, -1)
			linkData = linkData + link
		}
		listpageData = strings.Replace(listpageData, "{{content}}", linkData, -1) //insert these links into the list data

		fname := strconv.Itoa(i)
		if fname == "0" {
			fname = "index"
		}
		
		err = create_file(filepath.Join(output_directory, blog_subdirectory), fname + ".html", listpageData)
		fail_on_error(err)
	}
	return nil
}

func main() {
	args := os.Args[1:]
	input_directory := args[0]
	output_directory := args[1]

	input_post_directory := filepath.Join(input_directory, "_posts")
	input_static_directory := filepath.Join(input_directory, "_static")
	input_template_directory := filepath.Join(input_directory, "_templates")

	blog_subdirectory := "blog"

	//load every post file into a data structure
	posts, err := get_posts(input_post_directory, blog_subdirectory, 32)
	fail_on_error(err)

	//delete everything from the OUTPUT_DIRECTORY directory
	err = os.RemoveAll(output_directory)
	fail_on_error(err)

	//copy all static files to the output folder
	err = copy_directory(input_static_directory, output_directory)
	fail_on_error(err)

	//create a file for every post
	err = save_posts(posts, output_directory, input_template_directory)
	fail_on_error(err)

	//generate and save post lists
	err = createLists(posts, input_template_directory, output_directory, blog_subdirectory, 5)
	fail_on_error(err)
}
