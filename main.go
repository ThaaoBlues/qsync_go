package main

import (
	"bytes"
	. "fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"strconv"
)

/*
=====================================
DATABASE
=====================================

*/

// content is a struct which contains a file's name, its type and its data.

func sendPostRequest(url string, full_path string, sync_id string) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	//before all, check if update is not on a file that has been deleted right before
	if _, err := os.Stat(full_path); os.IsNotExist(err) {
		delete_file_from_db(sync_id, full_path)
		println("[+] Updating db after file deletion : " + full_path)
		return
	}

	// New multipart writer.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile("file", filepath.Base(full_path))
	if err != nil {
		println("Error while sending file : " + filepath.Base(full_path) + " : " + string(err.Error()))
	}
	file, err := os.Open(full_path)
	if err != nil {
		println("Error while sending file : " + string(err.Error()))
	}
	_, err = io.Copy(fw, file)
	if err != nil {
		println("Error while sending file : " + string(err.Error()))
	}
	writer.Close()
	req, err := http.NewRequest("POST", url, bytes.NewReader(body.Bytes()))
	if err != nil {
		println("Error while sending file : " + string(err.Error()))
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rsp, _ := client.Do(req)
	if rsp.StatusCode != http.StatusOK {
		log.Printf("Request failed with response code: %d "+filepath.Base(full_path), rsp.StatusCode)
	}

	file.Close()
}

func upload_file(full_path string, ip_addr string, sync_id string,sync_root string) {
	relative_path := strings.Replace(full_path,sync_root,"",1)

	is_local_second_end := is_local_second_end(sync_id,sync_root)

	if is_sync_local(sync_id){
		// to notify folder creation on the other local end and not on the same
		is_local_second_end = !is_local_second_end
	}

	url := "http://" + ip_addr + "/sync_upload?relative_path=" + url.QueryEscape(relative_path) + "&sync_id=" + sync_id +"&is_local_second_end="+strconv.FormatBool(is_local_second_end)
	sendPostRequest(url, full_path, sync_id)

}

func GetFileContentType(ouput *os.File) (string, error) {

	// to sniff the content type only the first
	// 512 bytes are used.

	buf := make([]byte, 512)

	_, err := ouput.Read(buf)

	if err != nil {
		return "", err
	}

	// the function that actually does the trick
	contentType := http.DetectContentType(buf)

	return contentType, nil
}

func gen_sync_id() string {
	rand.Seed(time.Now().UnixNano())
	sync_id_list := make([]string, 126)
	const char_list = string("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	for i := range sync_id_list {
		sync_id_list[i] = string(char_list[rand.Intn(len(char_list))])
	}
	sync_id := strings.Join(sync_id_list, "")
	return sync_id
}

func init_db(sync_root string, sync_id string, remote_addr string) {
	os.WriteFile(sync_id+"_files.csv", []byte("id;relative_path;m_date"), 0644)
	os.WriteFile(sync_id+"_folders.csv", []byte("id;relative_path"), 0644)


	is_local_second_end := false
	bdd_content, _ := os.ReadFile("sync_db.csv")
	bdd_content_split := strings.Split(string(bdd_content), "\n")

	for _, ele := range bdd_content_split {

		// already the first end in local database, so mark the new end as local_second_end
		if (strings.Split(ele, ";")[0] == sync_id){
			is_local_second_end = true
		}
	}

	

	dirs, _ := os.ReadDir(".")

	//adding sync_id into the database
	found := false
	for _, dir := range dirs {
		if dir.Name() == "sync_db.csv" {
			db_ctt, _ := os.ReadFile("sync_db.csv")
			os.WriteFile("sync_db.csv", []byte(string(db_ctt)+"\n"+sync_id+";"+sync_root+";"+remote_addr+";"+strconv.FormatBool(is_local_second_end)), 0644)
			found = true
			break
		}

	}

	// database is not created ?
	if !found {
		os.WriteFile("sync_db.csv", []byte("sync_id;sync_root;remote_addr;is_local_second_end"), 0644)
		db_ctt, _ := os.ReadFile("sync_db.csv")
		os.WriteFile("sync_db.csv", []byte(string(db_ctt)+"\n"+sync_id+";"+sync_root+";"+remote_addr+";"+strconv.FormatBool(is_local_second_end)), 0644)
	}

}

func get_remote_addr(sync_id string) string {

	db_ctt, _ := os.ReadFile("sync_db.csv")
	db_ctt_list := strings.Split(string(db_ctt), "\n")

	for _, ele := range db_ctt_list {

		line := strings.Split(ele, ";")

		if line[0] == sync_id {
			return line[2]
		}
	}

	return ""

}

func is_id_valid(sync_id string) bool {
	bdd_content, _ := os.ReadFile("sync_db.csv")
	bdd_content_split := strings.Split(string(bdd_content), "\n")

	for _, ele := range bdd_content_split {
		if strings.Split(ele, ";")[0] == sync_id {
			return true
		}
	}

	// id not valid
	return false
}


func is_local_second_end(sync_id string, sync_root string) bool{
	bdd_content, _ := os.ReadFile("sync_db.csv")
	bdd_content_split := strings.Split(string(bdd_content), "\n")

	for _, ele := range bdd_content_split {

		ele_split := strings.Split(ele, ";")
		// test if we are on the right sync_id by comparing path as this is the only thing changing
		if (ele_split[0] == sync_id) && (strings.Contains(sync_root,ele_split[1])) && (strings.Contains("true",ele_split[3])){
			return true
		}
	}

	return false

}

func is_sync_local(sync_id string) bool{
	bdd_content, _ := os.ReadFile("sync_db.csv")
	bdd_content_split := strings.Split(string(bdd_content), "\n")

	c := 0
	for _, ele := range bdd_content_split {

		// test if we are on the right sync_id by comparing path as this is the only thing changing
		if (strings.Split(ele, ";")[0] == sync_id){
			c += 1
		}
	}

	return c == 2
}

/*

function called in map_directory at each file encountered
*/
func register_file(full_path string, e os.DirEntry, sync_id string,sync_root string) {



	if needs_update(full_path, e, sync_id,sync_root) {

		relative_path := strings.Replace(full_path,sync_root,"",1)

		info, _ := e.Info()
		modtime := info.ModTime().Format(time.RFC1123)
		bdd_content, _ := os.ReadFile(sync_id + "_files.csv")

		if file_registered(relative_path, sync_id) {
			//update database just by replacing m_date
			var new_bdd_content string
			bdd_string_split := strings.Split(string(bdd_content), "\n")
			new_bdd_content = bdd_string_split[0]
			for _, ele := range bdd_string_split[1:] {

				// locate the right line
				if strings.Contains(ele, relative_path) {
					// get it and apply modification
					line_rest := strings.Join(strings.Split(ele, ";")[:2], ";")
					line_rest = "\n" + line_rest + ";" + modtime

					new_bdd_content = new_bdd_content + line_rest
				} else {
					new_bdd_content = new_bdd_content + "\n" + ele
				}

			}
			Println("[+] Modifying file : "+relative_path+" in database")
			// write database with modified line
			os.WriteFile(sync_id+"_files.csv", []byte(new_bdd_content), os.ModeAppend)
		} else {
			Println("[+] Adding file : "+relative_path+" to database")
			//update database by adding the whole line
			os.WriteFile(sync_id+"_files.csv", []byte(string(bdd_content)+"\n0;"+relative_path+";"+modtime), os.ModeAppend)

		}

		//notify update to remote and send file
		upload_file(full_path, get_remote_addr(sync_id), sync_id,sync_root)

	}
}

/*
function called in map_directory at each folder encountered
*/
func register_folder(full_path string, e os.DirEntry, sync_id string,sync_root string) {

	if needs_update(full_path, e, sync_id,sync_root) {

		relative_path := strings.Replace(full_path,sync_root,"",1)
		Println("Updating folders database with : " + relative_path)

		//notify update to remote
		client := http.Client{
			Timeout: time.Second / 10,
		}

		is_local_second_end := is_local_second_end(sync_id,sync_root)

		if is_sync_local(sync_id){
			// to notify folder creation on the other local end and not on the same
			is_local_second_end = !is_local_second_end
		}

		client.Get("http://" + get_remote_addr(sync_id) + "/create_folder?relative_path=" + url.QueryEscape(relative_path) + "&sync_id=" + sync_id+"&is_local_second_end="+strconv.FormatBool(is_local_second_end))
		
		//update database
		bdd_content, _ := os.ReadFile(sync_id + "_folders.csv")
		os.WriteFile(sync_id+"_folders.csv", []byte(string(bdd_content)+"\n0;"+relative_path), os.ModeAppend)

	}
}

func needs_update(full_path string, e os.DirEntry, sync_id string,sync_root string) bool {


	relative_path := strings.Replace(full_path,sync_root,"",1)
	if e.IsDir() {
		return !dir_registered(relative_path, sync_id)
	} else {
		if file_registered(relative_path, sync_id) {

			f := get_file(full_path,relative_path, sync_id)
			new_info, _ := e.Info()

			db_parsed_time, _ := time.Parse(time.RFC1123, f["m_date"])
			m_time := new_info.ModTime()

			// trigger update only if modification time is one minute after the one stored in db
			if m_time.After(db_parsed_time.Add(time.Minute * time.Duration(1))) {
				Println("updating file : " + relative_path + "\n\tdate diff :" + m_time.Format(time.RFC1123) + " vs " + f["m_date"])
				return true
			}
		} else {
			return true
		}
	}
	return false
}


func get_file(full_path string,relative_path string, sync_id string) map[string]string {
	bytes_bdd_content, _ := os.ReadFile(sync_id + "_files.csv")
	str_bdd_content := strings.Split(string(bytes_bdd_content), "\n")
	
	//file already being written by another process, wait a bit
	if len(str_bdd_content) <= 1{
		time.Sleep(1*time.Second)
		return get_file(full_path,relative_path,sync_id)
	}

	for _, ele := range str_bdd_content {
		file_info := strings.Split(string(ele), ";")
		if file_info[1] == relative_path {
			ret := map[string]string{"full_path": file_info[1], "m_date": file_info[2]}
			return ret
		}
	}
	return map[string]string{"full_path": "", "m_date": ""}
}

func file_registered(relative_path string, sync_id string) bool {
	bytes_bdd_content, _ := os.ReadFile(sync_id + "_files.csv")
	str_bdd_content := strings.Split(string(bytes_bdd_content), "\n")
	for _, ele := range str_bdd_content {


		// may occur in a context of local to local sync
		if (len(strings.Split(string(ele), ";"))> 1){
			if strings.Split(string(ele), ";")[1] == relative_path {
				return true
			}
		}


	}
	return false
}

func dir_registered(relative_path string, sync_id string) bool {
	bytes_bdd_content, _ := os.ReadFile(sync_id + "_folders.csv")
	str_bdd_content := strings.Split(string(bytes_bdd_content), "\n")

	for _, ele := range str_bdd_content {

		// may occur in a context of local to local sync
		if (len(strings.Split(string(ele), ";"))> 1){

			if strings.Split(string(ele), ";")[1] == relative_path {
				return true
			}

		}
	}

	return false
}

func map_directory(directory string, sync_id string,sync_root string) {
	entries, err := os.ReadDir(directory)

	if err != nil {
		// directory must have been delete, proceed to remove it from db
		check_files_deletion(sync_root,sync_id)
	}

	for _, e := range entries {
		full_path := directory + "/" + e.Name()
		if e.IsDir() {
			register_folder(full_path, e, sync_id,sync_root)
			map_directory(full_path, sync_id,sync_root)
		} else {
			register_file(full_path, e, sync_id,sync_root)
		}

	}

	

}


func check_files_deletion(sync_root string, sync_id string){
	// now, check for files deletion
	bytes_bdd_content, _ := os.ReadFile(sync_id + "_files.csv")
	str_bdd_content := strings.Split(string(bytes_bdd_content), "\n")[1:]
	for _, ele := range str_bdd_content {

		// true if not found error
		file_relative_path := strings.Split(string(ele), ";")[1]
		if _, err := os.Stat(sync_root+file_relative_path); os.IsNotExist(err) {
			delete_file_from_db(sync_id, file_relative_path)

			//notify other end
			notify_file_deletion(sync_id,get_remote_addr(sync_id),file_relative_path,sync_root)
			println("[+] Updating db after file deletion : " + file_relative_path)

		}
	}

	// now, check for folders deletion
	bytes_bdd_content, _ = os.ReadFile(sync_id + "_folders.csv")
	str_bdd_content = strings.Split(string(bytes_bdd_content), "\n")[1:]
	for _, ele := range str_bdd_content {

		// true if not found error
		folder_relative_path := strings.Split(string(ele), ";")[1]
		if _, err := os.Stat(sync_root+folder_relative_path); os.IsNotExist(err) {
			delete_folder_from_db(sync_id, folder_relative_path)
			//notify other end
			notify_folder_deletion(sync_id,get_remote_addr(sync_id),folder_relative_path,sync_root)

			println("[+] Updating db after folder deletion : " + folder_relative_path)
		}
	}
}

func delete_file_from_db(sync_id string, relative_path string) {
	bdd_content, _ := os.ReadFile(sync_id + "_files.csv")
	var new_bdd_content string
	bdd_string_split := strings.Split(string(bdd_content), "\n")

	//file is already being written
	if len(bdd_string_split) <= 1{
		time.Sleep(1*time.Second)
		delete_file_from_db(sync_id, relative_path)
		return
	}

	//file is already being written
	if len(bdd_string_split) <= 1{
		time.Sleep(1*time.Second)
		delete_file_from_db(sync_id, relative_path)
		return
	}

	new_bdd_content = bdd_string_split[0]
	for _, ele := range bdd_string_split[1:] {

		// locate all except the right line
		if !strings.Contains(ele, relative_path) {
			new_bdd_content = new_bdd_content + "\n" + ele
		}

	}

	// write database without delete file's line
	os.WriteFile(sync_id+"_files.csv", []byte(new_bdd_content), os.ModeAppend)
}

func delete_folder_from_db(sync_id string, relative_path string) {
	bdd_content, _ := os.ReadFile(sync_id + "_folders.csv")
	var new_bdd_content string
	bdd_string_split := strings.Split(string(bdd_content), "\n")

	//file is already being written
	if len(bdd_string_split) <= 1{
		time.Sleep(1*time.Second)
		delete_folder_from_db(sync_id, relative_path)
		return
	}

	new_bdd_content = bdd_string_split[0]
	for _, ele := range bdd_string_split[1:] {

		// add all except the right line
		if !strings.Contains(ele, relative_path) {
			new_bdd_content = new_bdd_content + "\n" + ele
			
		}

	}

	// write database with modified line
	os.WriteFile(sync_id+"_folders.csv", []byte(new_bdd_content), os.ModeAppend)
}

func notify_folder_creation(relative_path string, ip_addr string, sync_id string,sync_root string) {

	is_local_second_end := is_local_second_end(sync_id,sync_root)

	client := http.Client{
		Timeout: time.Second / 10,
	}
	client.Get("http://" + ip_addr + "/create_folder?relative_path=" + url.QueryEscape(relative_path) + "&sync_id=" + sync_id+"&is_local_second_end="+strconv.FormatBool(is_local_second_end))
}

func notify_folder_deletion(sync_id string, ip_addr string,relative_path string,sync_root string) {

	is_local_second_end := is_local_second_end(sync_id,sync_root)

	if is_sync_local(sync_id){
		// to notify folder creation on the other local end and not on the same
		is_local_second_end = !is_local_second_end
	}
	client := http.Client{
		Timeout: time.Second / 10,
	}
	client.Get("http://" + ip_addr + "/delete_folder?relative_path=" + url.QueryEscape(relative_path) + "&sync_id=" + sync_id+"&is_local_second_end="+strconv.FormatBool(is_local_second_end))
}

func notify_file_deletion(sync_id string, ip_addr string,relative_path string,sync_root string) {
	is_local_second_end := is_local_second_end(sync_id,sync_root)

	if is_sync_local(sync_id){
		// to notify folder creation on the other local end and not on the same
		is_local_second_end = !is_local_second_end
	}

	client := http.Client{
		Timeout: time.Second / 10,
	}
	client.Get("http://" + ip_addr + "/delete_file?relative_path=" + url.QueryEscape(relative_path) + "&sync_id=" + sync_id+"&is_local_second_end="+strconv.FormatBool(is_local_second_end))
}

func restart_tasks() {
	db_ctt, _ := os.ReadFile("sync_db.csv")
	db_ctt_list := strings.Split(string(db_ctt), "\n")
	for _, ele := range db_ctt_list[1:] {
		ele_split := strings.Split(ele, ";")
		go sync_process(ele_split[0], ele_split[1])

		//don't start all tasks at the same time so we avoid misswriting or new folder deletion
		time.Sleep(1*time.Second)

		println("\t[+] task root : " + ele_split[1])
	}

	println("[v] All sync tasks are now active.")
}

func update_at_creation(sync_id string, ip_addr string,sync_root string) {
	db_ctt, _ := os.ReadFile("sync_db.csv")
	db_ctt_list := strings.Split(string(db_ctt), "\n")
	for _, ele := range db_ctt_list[1:] {

		ele_split := strings.Split(ele, ";")

		if ele_split[0] == sync_id {

			println("\t\t[+] Seeking remote updates for this task...")

			//found the right sync_id, now get all files from the other end

			// getting files db

			files_resp, err := http.Get("http://" + ip_addr + "/sync_db/files?sync_id=" + sync_id)

			if err != nil {
				println("[!] Error while creating task : Other end device offline")
			}

			files_db_ctt, _ := io.ReadAll(files_resp.Body)
			os.WriteFile(sync_id+"_files.csv", files_db_ctt, 0644)

			db_ctt, _ := os.ReadFile(sync_id+"_files.csv")
			println("files.csv content : "+string(db_ctt))

			println("\t\t[+] Downloaded other end files database")

			// getting folders db

			folders_resp, err := http.Get("http://" + ip_addr + "/sync_db/folders?sync_id=" + sync_id)

			if err != nil {
				println("[!] Error while creating task : Other end device offline")
				return
			}

			folders_db_ctt, _ := io.ReadAll(folders_resp.Body)
			os.WriteFile(sync_id+"_folders.csv", folders_db_ctt, 0644)

			println("\t\t[+] Downloaded other end folders database")

			// now create all folders

			folders_db_ctt_split := strings.Split(string(folders_db_ctt), "\n")
			for _, ele := range folders_db_ctt_split {

				full_path := sync_root + strings.Split(ele, ";")[1]

				os.Mkdir(full_path, os.ModeDir)

			}

			println("\t\t[+] Created all folders")

			//now download all files

			files_db_ctt_split := strings.Split(string(files_db_ctt), "\n")
			for _, ele := range files_db_ctt_split {

				full_path := sync_root + strings.Split(ele, ";")[1]

				resp, _ := http.Get("http://" + ip_addr + "/download?full_path=" + url.QueryEscape(full_path) + "&sync_id=" + sync_id)
				file_ctt, _ := io.ReadAll(resp.Body)
				os.WriteFile(full_path, file_ctt, 0644)
			}

			println("\t\t[+] Created all files")

			break

		}
	}

}

func get_sync_root(sync_id string,is_local_second_end bool) string {
	db_ctt, _ := os.ReadFile("sync_db.csv")
	db_ctt_list := strings.Split(string(db_ctt), "\n")
	for _, ele := range db_ctt_list[1:] {
		ele_split := strings.Split(ele, ";")

		if (ele_split[0] == sync_id) && (ele_split[3] == strconv.FormatBool(is_local_second_end)) {
			// found the right sync task, return its root
			return ele_split[1]
		}
	}

	return ""
}

/*
Sync process is the function started as a coroutine that will loop map_directory() indefinitely
*/
func sync_process(sync_id string, directory string) {

	for true {
		// slow down the loop and keep unsynchronised sync tasks at the same time
		// to avoid misswrite and new folder deletion
		time.Sleep(5*time.Second)


		map_directory(directory, sync_id,directory)
		check_files_deletion(directory,sync_id)
	}
}

/*
=========================================
WEB SERVER
=========================================
*/

type sync_task struct{
	Sync_id string
	Sync_root string
	Remote_addr string
	Is_local_second_end string
}

func main() {

	Println("[+] Starting server...")

	//check if main db is created

	dirs, _ := os.ReadDir(".")

	found := false
	for _, dir := range dirs {
		if dir.Name() == "sync_db.csv" {
			found = true
			break
		}

	}

	// database is not created ?
	if !found {
		os.WriteFile("sync_db.csv", []byte("sync_id;sync_root;remote_addr;is_local_second_end"), 0644)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// accept only local requests on this endpoint
		if strings.Contains(r.RemoteAddr, ".") {
			w.WriteHeader(403)
			http.Error(w,"Access forbidden to this url :/",403)
			return
		}


		t, _ := template.ParseFiles("templates/index.html")
		db_ctt, _ := os.ReadFile("sync_db.csv")

		db_ctt_list := strings.Split(string(db_ctt), "\n")
		
		var db []sync_task

		for _, ele := range db_ctt_list[1:] {
			ele_split := strings.Split(ele, ";")
			db = append(db,sync_task{Sync_id : ele_split[0], Sync_root : ele_split[1],Remote_addr : ele_split[2], Is_local_second_end : ele_split[3]})
		}
		

		t.Execute(w, db)

	})

	http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {

		// accept only local requests on this endpoint
		if strings.Contains(r.RemoteAddr, ".") {
			w.WriteHeader(403)
			http.Error(w,"Access forbidden to this url :/",403)
			return
		}

		sync_id := r.URL.Query().Get("sync_id")
		sync_root := r.URL.Query().Get("sync_root")

		remote_addr := r.URL.Query().Get("remote_addr")

		if (remote_addr == "localhost") || (remote_addr == "127.0.0.1") {
			//adding sync_id into the database
			db_ctt, _ := os.ReadFile("sync_db.csv")
			os.WriteFile("sync_db.csv", []byte(string(db_ctt)+"\n"+sync_id+";"+sync_root+";"+remote_addr+";"+"true"), 0644)
		} else {
			init_db(sync_root, sync_id, remote_addr)
		}

		// init sync database for this task and download files from the other end
		update_at_creation(sync_id, remote_addr,sync_root)

		// start goroutine
		go sync_process(sync_id, sync_root)

		http.Redirect(w, r, "/", 200)

	})

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {

		sync_id := r.URL.Query().Get("sync_id")
		full_path := r.URL.Query().Get("full_path")

		if !is_id_valid(sync_id) {
			http.Error(w, "Invalid sync_id", 404)
		}

		if !strings.Contains(full_path, get_sync_root(sync_id,is_local_second_end(sync_id,full_path))) {
			http.Error(w, "Invalid path", 404)
		}

		http.ServeFile(w, r, full_path)

	})

	// to update db after being notified that the other end have a newer version
	http.HandleFunc("/sync_db/files", func(w http.ResponseWriter, r *http.Request) {

		sync_id := r.URL.Query().Get("sync_id")
		if !is_id_valid(sync_id) {
			http.Error(w, "Invalid sync_id", 404)
		}
		db_content, _ := os.ReadFile(sync_id + "_files.csv")
		Fprintf(w, string(db_content))
	})

	http.HandleFunc("/sync_db/folders", func(w http.ResponseWriter, r *http.Request) {
		sync_id := r.URL.Query().Get("sync_id")
		if !is_id_valid(sync_id) {
			http.Error(w, "Invalid sync_id", 404)
		}
		db_content, _ := os.ReadFile(sync_id + "_folders.csv")
		Fprintf(w, string(db_content))
	})

	//upload one/multiples files on this machine ( will trigger a request of sync_db update)
	http.HandleFunc("/sync_upload", func(w http.ResponseWriter, r *http.Request) {

		sync_id := r.URL.Query().Get("sync_id")
		is_local_second_end := (r.URL.Query().Get("is_local_second_end") == "true")
		full_path := get_sync_root(sync_id,is_local_second_end) + r.URL.Query().Get("relative_path")

		println("[+] Downloading file upload : "+full_path+"\n\trelative path : "+r.URL.Query().Get("relative_path"))

		err := r.ParseMultipartForm(32 << 20) // maxMemory 32MB
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		tmpfile, err := os.Create(full_path)
		defer tmpfile.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = io.Copy(tmpfile, file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(200)

		var ip_addr string
		if !strings.Contains(r.RemoteAddr, ".") {
			ip_addr = "localhost"
		} else {
			ip_addr = r.RemoteAddr
		}

		client := http.Client{
			Timeout: time.Second / 10,
		}
		res, err := client.Get("http://" + ip_addr + "/sync_db/files?sync_id=" + sync_id)

		if err != nil {
			println("Erreur dans client.Get() : remote_addr=" + ip_addr + "/ err=" + string(err.Error()))
			return
		}

		files_db, _ := io.ReadAll(res.Body)

		os.WriteFile(sync_id+"_files.csv", files_db, os.ModeAppend)

	})

	//this endpoint will trigger a folder creation on this machine
	http.HandleFunc("/create_folder", func(w http.ResponseWriter, r *http.Request) {		
		sync_id := r.URL.Query().Get("sync_id")
		is_local_second_end := (r.URL.Query().Get("is_local_second_end") == "true")
		full_path := get_sync_root(sync_id,is_local_second_end) + r.URL.Query().Get("relative_path")
		os.Mkdir(full_path, os.ModeDir)


		Println("Updating folders database with : " + full_path+"\n\trelative path : "+r.URL.Query().Get("relative_path"))
		//update database
		bdd_content, _ := os.ReadFile(sync_id + "_folders.csv")
		os.WriteFile(sync_id+"_folders.csv", []byte(string(bdd_content)+"\n0;"+r.URL.Query().Get("relative_path")), os.ModeAppend)

	})
	//this endpoint will trigger a folder deletion on this machine
	http.HandleFunc("/delete_folder", func(w http.ResponseWriter, r *http.Request) {		
		sync_id := r.URL.Query().Get("sync_id")
		is_local_second_end := (r.URL.Query().Get("is_local_second_end") == "true")

		full_path := get_sync_root(sync_id,is_local_second_end) + r.URL.Query().Get("relative_path")

		Println("[+] Deleting folder in database : " + full_path + "\n\trelative_path : "+r.URL.Query().Get("relative_path"))
		//update database
		delete_folder_from_db(sync_id,r.URL.Query().Get("relative_path"))
		os.RemoveAll(full_path)

	})

	//this endpoint will trigger a folder deletion on this machine
	http.HandleFunc("/delete_file", func(w http.ResponseWriter, r *http.Request) {		
		sync_id := r.URL.Query().Get("sync_id")
		is_local_second_end := (r.URL.Query().Get("is_local_second_end") == "true")
		full_path := get_sync_root(sync_id,is_local_second_end) + r.URL.Query().Get("relative_path")

		Println("[+] Deleting file in database : " + full_path)
		//update database
		delete_file_from_db(sync_id,r.URL.Query().Get("relative_path"))
		os.Remove(full_path)

	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {


		// accept only local requests on this endpoint
		if strings.Contains(r.RemoteAddr, ".") {
			w.WriteHeader(403)
			http.Error(w,"Access forbidden to this url :/",403)
			return
		}

		full_path := r.URL.Query().Get("full_path")
		remote_addr := r.URL.Query().Get("remote_addr")

		if (full_path == "") || (remote_addr == ""){
			http.Error(w,"Empty form arguments, <a href='/'>Home</a>",404)
		}

		// init sync database for this task
		sync_id := gen_sync_id()
		init_db(full_path, sync_id, remote_addr)

		// start goroutine
		go sync_process(sync_id, full_path)

		http.Redirect(w, r, "/", 200)

	})


	println("[+] Restarting all tasks...")

	restart_tasks()

	println("[+] Starting server on http://localhost")

	http.ListenAndServe(":80", nil)

}
