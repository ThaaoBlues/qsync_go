# QSYNC_GO

## /!\ this software is not finished and comports many bugs that may lead to unexpected behaviors !
## qysync_go is a two ways folders synchronisation software entirely written in go and inspired by my last attempt in python ( very messy )

### this version is better for three main reasons :

- faster as golang is faster than python
- can be compiled and run on any OS
- the process used to sync folders is way more reliable and simple


### How it works ?

- both sides of the synchronisation task have a local database indexing files and folders that we want to sync
- when a difference is detected between the local database and the real folder content :
    - qsync notifies the other end by making a request to upload/delete what we want
    - the other end gets the request,makes whatever is needed on its side and finishes by making a request to the first end to get its database content and replace the outdated one

- each sync task is based on an infinite loop that is executed each 5 seconds
    - this loop recursively scan the desired folder (and makes comparisons with local database etc...)

- each task process will be restarted when you start qsync (but stay paused if it is)

### Special behaviors/additionnal informations : 

- in each task loop, before scanning the folder, qsync makes sure the other end is available. If not, it will automatically pause the task.

- if you want to have only a one-way synchronisation (like a copy), just go to the web interface at http://localhost:9214/ and pause the task (it will not pause the other end). But you still needs qsync running so it listen for updates coming from the other end.

- on linux, you may need to run this a root. (I experienced permission errors on KUbuntu)