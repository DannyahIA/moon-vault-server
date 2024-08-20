package controller

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DannyahIA/filemanager"
	"github.com/gin-gonic/gin"
)

const dir = "./drive"

var Files []filemanager.File
var FolderFiles []filemanager.File = []filemanager.File{}

func configFileManager() {
	filemanager.DefaultRoot = dir
}

func ListRootFoldersHandler(c *gin.Context) {
	configFileManager()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	files, err := filemanager.GetRootFolders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	Files = files

	if Files != nil {
		c.JSON(http.StatusOK, Files)
	} else {
		c.JSON(http.StatusOK, []filemanager.File{})
	}
}

func ListFilesHandler(c *gin.Context) {
	var requestBody struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	path := requestBody.Path
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query parameter is required"})
		return
	}

	var err error
	FolderFiles, err = filemanager.GetFolderItems(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if FolderFiles != nil {
		c.JSON(http.StatusOK, FolderFiles)
	} else {
		c.JSON(http.StatusOK, []filemanager.File{})
	}
}

func CreateFolderHandler(c *gin.Context) {
	type Request struct {
		Folder string 
		Path   string
	}
	
	request := Request{}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Folder == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "folder parameter is required"})
		return
	}

	if err := filemanager.CreateFolder(request.Path, request.Folder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func DeleteFileHandler(c *gin.Context) {
	var requestBody []struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, delete := range requestBody {
		path := delete.Path

		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path query parameter is required"})
			return
		}

		for _, file := range FolderFiles {
			if file.Path == path {
				if err := filemanager.DeleteItem(file.IsFolder, path); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
	}
	c.Status(http.StatusOK)
}

func UploadFileHandler(c *gin.Context) {
	const maxFileSize = 50000 /*<- 50000mb*/ << 20

	err := c.Request.ParseMultipartForm(maxFileSize)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	file, handler, err := c.Request.FormFile("file")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	defer file.Close()

	path := c.Request.FormValue("path")
	filePath := filepath.Join(".", path, handler.Filename)
	for _, f := range FolderFiles {
		if filepath.Join(f.Path) == filePath {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "There is already a file with the same name in this location."})
			return
		}
	}

	err = c.SaveUploadedFile(handler, filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensagem": "File uploaded"})
}

func DownloadFileHandler(c *gin.Context) {
	var requestBody struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	path := requestBody.Path
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query parameter is required"})
		return
	}

	c.File(path)
}

func RenameFileHandler(c *gin.Context) {
	var requestBody []struct {
		Path     string `json:"path"`
		NewName  string `json:"new_name"`
		IsFolder bool   `json:"is_folder"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, req := range requestBody {
		if len(requestBody) > 1 {
			ext := filepath.Ext(req.NewName)
			req.NewName = req.NewName[:len(req.NewName)-len(ext)] + filepath.Ext(req.Path)
		}
		path := req.Path
		newName := req.NewName
		isFolder := req.IsFolder

		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path query parameter is required"})
			return
		}

		if newName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "newName query parameter is required"})
			return
		}

		exists := filemanager.Exists(path)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "File or folder not found"})
			return
		}

		dir, base := filepath.Split(path)

		newPath := filepath.Join(dir, newName)
		if isFolder {
			newPath = filepath.Join(newPath, base)
		}
		exists = filemanager.Exists(newPath)
		if exists {
			i := 1
			for {
				newNameWithNumber := fmt.Sprintf("%s (%d)", newName[:len(newName)-len(filepath.Ext(newName))], i)
				if isFolder {
					newNameWithNumber = filepath.Join(newNameWithNumber, base)
				}
				if filepath.Ext(newName) != "" && filepath.Ext(newNameWithNumber) == "" {
					newNameWithNumber += filepath.Ext(newName)
				}
				newPathWithNumber := filepath.Join(dir, newNameWithNumber)
				exists = filemanager.Exists(newPathWithNumber)
				if !exists {
					newPath = newPathWithNumber
					break
				}
				i++
			}
		}

		err := filemanager.Rename(path, newPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Files renamed successfully"})
}

func SearchFileHandler(c *gin.Context) {
	var requestBody struct {
		Query string `json:"query"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := requestBody.Query
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query query parameter is required"})
		return
	}

	files, err := filemanager.Search(strings.ToLower(query))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if files != nil {
		c.JSON(http.StatusOK, files)
	} else {
		c.JSON(http.StatusOK, []filemanager.File{})
	}
}
