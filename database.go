package usb_share

import (
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

/*
=====================================
DATABASE
=====================================

*/
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
	os.WriteFile(sync_id+"files.csv", []byte("id;full_path;m_date"), 0644)
	os.WriteFile(sync_id+"folders.csv", []byte("id;full_path"), 0644)

	dirs, _ := os.ReadDir(".")

	//adding sync_id into the database
	found := false
	for _, dir := range dirs {
		if dir.Name() == "sync_db.csv" {
			db_ctt, _ := os.ReadFile("sync_db.csv")
			os.WriteFile("sync_db.csv", []byte(string(db_ctt)+"\n"+sync_id+";"+sync_root), 0644)
			found = true
			break
		}

	}

	// database is not created ?
	if !found {
		os.WriteFile("sync_db.csv", []byte("sync_id;sync_root;remote_addr;\n"), 0644)
		db_ctt, _ := os.ReadFile("sync_db.csv")
		os.WriteFile("sync_db.csv", []byte(string(db_ctt)+"\n"+sync_id+";"+sync_root+";"+remote_addr+";"+"\n"), 0644)
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

func register_file(full_path string, e os.DirEntry, sync_id string) {

	if needs_update(full_path, e, sync_id) {
		Println("Updating files database with : " + full_path)

		//update database
		info, _ := e.Info()
		modtime := info.ModTime().Format(time.RFC1123)
		bdd_content, _ := os.ReadFile("files.csv")
		os.WriteFile(sync_id+"_files.csv", []byte(string(bdd_content)+"\n0;"+full_path+";"+modtime), os.ModeAppend)

		//notify update to remote and send file
		upload_file(full_path, get_remote_addr(sync_id), sync_id)

	}
}

func register_folder(full_path string, e os.DirEntry, sync_id string) {
	if needs_update(full_path, e, sync_id) {
		Println("Updating folders database with : " + full_path)
		//update database
		bdd_content, _ := os.ReadFile(sync_id + "_folders.csv")
		os.WriteFile(sync_id+"_folders.csv", []byte(string(bdd_content)+"\n0;"+full_path), os.ModeAppend)

		//notify update to remote
		http.Get("http://" + get_remote_addr(sync_id) + "/sync_folder?full_path=" + url.QueryEscape(full_path) + "&sync_id=" + sync_id)

	}

}

func needs_update(full_path string, e os.DirEntry, sync_id string) bool {
	if e.IsDir() {
		return !dir_registered(full_path, sync_id)
	} else {
		if file_registered(full_path, sync_id) {
			f := get_file(full_path, sync_id)
			new_info, _ := e.Info()

			db_parsed_time, _ := time.Parse(time.RFC1123, f["m_date"])

			if new_info.ModTime().After(db_parsed_time) {
				Println("AFTER")
				Println(new_info.ModTime().Format(time.RFC1123) + " vs " + f["m_date"])
				return true
			}
		} else {
			return true
		}
	}
	return false
}

func get_file(full_path string, sync_id string) map[string]string {
	bytes_bdd_content, _ := os.ReadFile(sync_id + "_files.csv")
	str_bdd_content := strings.Split(string(bytes_bdd_content), "\n")

	for _, ele := range str_bdd_content {
		file_info := strings.Split(string(ele), ";")
		if file_info[1] == full_path {
			ret := map[string]string{"full_path": file_info[1], "m_date": file_info[2]}
			return ret
		}
	}
	return map[string]string{"full_path": "", "m_date": ""}
}

func file_registered(full_path string, sync_id string) bool {
	bytes_bdd_content, _ := os.ReadFile(sync_id + "_files.csv")
	str_bdd_content := strings.Split(string(bytes_bdd_content), "\n")

	for _, ele := range str_bdd_content {

		if strings.Split(string(ele), ";")[1] == full_path {
			return true
		}
	}
	return false
}

func dir_registered(full_path string, sync_id string) bool {
	bytes_bdd_content, _ := os.ReadFile(sync_id + "_folders.csv")
	str_bdd_content := strings.Split(string(bytes_bdd_content), "\n")

	for _, ele := range str_bdd_content {
		if strings.Split(string(ele), ";")[1] == full_path {
			return true
		}
	}

	return false
}

func map_directory(directory string, sync_id string) {

	entries, _ := os.ReadDir(directory)

	for _, e := range entries {

		full_path := directory + "/" + e.Name()
		if e.IsDir() {
			map_directory(full_path, sync_id)
			register_folder(full_path, e, sync_id)
		} else {
			register_file(full_path, e, sync_id)
		}

	}

}

func notify_folder_creation(full_path string, ip_addr string, sync_id string) {
	http.Get("http://" + ip_addr + "/create_folder?full_path=" + url.QueryEscape(full_path))
}
