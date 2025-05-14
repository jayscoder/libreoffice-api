package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// 全局配置变量
var (
	DEBUG             bool
	MAX_CONTENT_LENGTH int64
	SOFFICE_PATH      string
	FILE_EXPIRY_HOURS int
	BASE_DIR          string
	TMP_DIR           string
	DATA_DIR          string

	libreofficeAvailable bool
	libreofficeVersion   string
)

// APIConfig 存储API的配置信息
type APIConfig struct {
	Debug           bool   `json:"debug"`
	MaxContentSize  int64  `json:"max_content_size"`
	SofficePath     string `json:"soffice_path"`
	FileExpiryHours int    `json:"file_expiry_hours"`
	DataDir         string `json:"data_dir"`
	TmpDir          string `json:"tmp_dir"`
}

// InitConfig 初始化配置
func InitConfig() {
	// 加载.env文件
	_ = godotenv.Load()

	// 设置默认值并从环境变量获取配置
	DEBUG = os.Getenv("DEBUG") != "false"
	
	// 文件大小默认100MB
	maxSizeStr := os.Getenv("MAX_CONTENT_LENGTH")
	if maxSizeStr == "" {
		MAX_CONTENT_LENGTH = 100 * 1024 * 1024
	} else {
		maxSize, err := strconv.ParseInt(maxSizeStr, 10, 64)
		if err != nil {
			MAX_CONTENT_LENGTH = 100 * 1024 * 1024
		} else {
			MAX_CONTENT_LENGTH = maxSize
		}
	}

	// LibreOffice 路径
	SOFFICE_PATH = os.Getenv("SOFFICE_PATH")
	if SOFFICE_PATH == "" {
		SOFFICE_PATH = "soffice"
	}

	// 文件过期时间
	fileExpiryStr := os.Getenv("FILE_EXPIRY_HOURS")
	if fileExpiryStr == "" || fileExpiryStr == "-1" {
		FILE_EXPIRY_HOURS = -1 // -1表示永不过期
	} else {
		expiryHours, err := strconv.Atoi(fileExpiryStr)
		if err != nil {
			FILE_EXPIRY_HOURS = 24 // 默认24小时
		} else {
			FILE_EXPIRY_HOURS = expiryHours
		}
	}

	// 设置目录
	BASE_DIR, _ = os.Getwd()
	TMP_DIR = filepath.Join(BASE_DIR, "tmp")
	DATA_DIR = filepath.Join(BASE_DIR, "data")

	// 创建必要的目录
	os.MkdirAll(TMP_DIR, 0755)
	os.MkdirAll(DATA_DIR, 0755)

	// 检查LibreOffice是否可用
	libreofficeAvailable, libreofficeVersion = checkLibreOffice()
	
	log.Printf("配置初始化完成: DEBUG=%v, MAX_CONTENT_LENGTH=%d, SOFFICE_PATH=%s, FILE_EXPIRY_HOURS=%d",
		DEBUG, MAX_CONTENT_LENGTH, SOFFICE_PATH, FILE_EXPIRY_HOURS)
}

// 检查LibreOffice是否可用
func checkLibreOffice() (bool, string) {
	cmd := exec.Command(SOFFICE_PATH, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("LibreOffice检测失败: %v", err)
		return false, err.Error()
	}
	version := strings.TrimSpace(string(output))
	log.Printf("检测到LibreOffice: %s", version)
	return true, version
}

// 定时清理过期文件
func startCleanupScheduler(ctx context.Context, wg *sync.WaitGroup) {
	if FILE_EXPIRY_HOURS <= 0 {
		log.Println("文件永不过期，不启动清理任务")
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		log.Printf("已启动文件清理任务，过期时间: %d小时", FILE_EXPIRY_HOURS)

		for {
			select {
			case <-ticker.C:
				log.Println("开始执行文件清理任务")
				cleanupExpiredFiles()
			case <-ctx.Done():
				log.Println("文件清理任务退出")
				return
			}
		}
	}()
}

// 清理过期文件
func cleanupExpiredFiles() {
	if FILE_EXPIRY_HOURS <= 0 {
		log.Println("文件永不过期，跳过清理")
		return
	}

	now := time.Now()
	expiryDuration := time.Duration(FILE_EXPIRY_HOURS) * time.Hour

	// 递归遍历目录
	err := filepath.Walk(DATA_DIR, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查文件是否过期
		fileTime := info.ModTime()
		if now.Sub(fileTime) > expiryDuration {
			if err := os.Remove(path); err != nil {
				log.Printf("删除过期文件时出错: %v", err)
			} else {
				log.Printf("已删除过期文件: %s", path)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("清理过期文件时出错: %v", err)
	}

	// 删除空目录
	removeEmptyDirs(DATA_DIR)
}

// 删除空目录
func removeEmptyDirs(root string) {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过根目录和非目录
		if !info.IsDir() || path == root {
			return nil
		}

		// 检查目录是否为空
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Readdirnames(1)
		if err == io.EOF {
			// 目录为空，删除
			if err := os.Remove(path); err != nil {
				log.Printf("删除空目录时出错: %v", err)
			} else {
				log.Printf("已删除空目录: %s", path)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("清理空目录时出错: %v", err)
	}
}

// 生成输出文件路径
func generateOutputFilepath(originalFilename, targetExt string) (string, string) {
	// 按日期生成目录
	dateStr := time.Now().Format("20060102")
	dateDir := filepath.Join(DATA_DIR, dateStr)

	// 确保日期目录存在
	os.MkdirAll(dateDir, 0755)

	// 提取原始文件名（不含扩展名）
	baseName := strings.TrimSuffix(filepath.Base(originalFilename), filepath.Ext(originalFilename))
	// 生成时间戳后缀
	timestampSuffix := time.Now().UnixNano() / int64(time.Millisecond)

	// 构建输出文件名
	outputFilename := fmt.Sprintf("%s_%d.%s", baseName, timestampSuffix, targetExt)
	outputFilepath := filepath.Join(dateDir, outputFilename)

	// 相对路径（用于构建URL）
	relativePath := filepath.Join(dateStr, outputFilename)

	return outputFilepath, relativePath
}

// ConversionResponse 转换结果响应
type ConversionResponse struct {
	Success         bool   `json:"success"`
	Filename        string `json:"filename"`
	DownloadURL     string `json:"download_url"`
	DownloadFilename string `json:"download_filename"`
	Text            string `json:"text,omitempty"`
	Expiry          string `json:"expiry"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status         string `json:"status"`
	LibreOffice    bool   `json:"libreoffice"`
	Version        string `json:"version"`
	DataDir        string `json:"data_dir"`
	FileExpiryHours int    `json:"file_expiry_hours"`
}

func main() {
	// 初始化配置
	InitConfig()
	
	// 创建上下文和WaitGroup，用于优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	var wg sync.WaitGroup
	
	// 启动定时清理任务
	startCleanupScheduler(ctx, &wg)
	
	// 设置Gin模式
	if !DEBUG {
		gin.SetMode(gin.ReleaseMode)
	}
	
	router := gin.Default()
	
	// 设置最大multipart表单内存大小
	router.MaxMultipartMemory = MAX_CONTENT_LENGTH
	
	// 设置API路由
	router.GET("/", indexHandler)
	router.GET("/health", healthCheckHandler)
	router.POST("/convert", convertDocumentHandler)
	router.GET("/download/:filename", downloadFileHandler)
	
	// 启动服务器
	log.Printf("启动服务: host=0.0.0.0, port=15000, debug=%v", DEBUG)
	log.Printf("文件存储目录: %s, 过期时间: %v 小时", DATA_DIR, 
		func() interface{} {
			if FILE_EXPIRY_HOURS <= 0 {
				return "永不过期"
			}
			return FILE_EXPIRY_HOURS
		}())
	
	server := &http.Server{
		Addr:    ":15000",
		Handler: router,
	}
	
	// 在goroutine中启动服务器
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务失败: %v", err)
		}
	}()
	
	// 等待中断信号
	quit := make(chan os.Signal, 1)
	// kill (无参数) 默认发送 syscall.SIGTERM
	// kill -2 发送 syscall.SIGINT
	// kill -9 发送 syscall.SIGKILL，但不能被捕获，所以不需要添加
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// 阻塞直到接收到退出信号
	sig := <-quit
	log.Printf("接收到退出信号: %v", sig)
	
	// 创建一个5秒超时的上下文
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 优雅地关闭服务器
	log.Println("关闭服务器...")
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("服务器关闭异常: %v", err)
	}
	
	// 等待所有goroutine完成
	log.Println("等待清理任务完成...")
	wg.Wait()
	log.Println("服务已完全关闭")
}

// 首页处理
func indexHandler(c *gin.Context) {
	html := `
    <html>
        <head>
            <title>LibreOffice文档转换API</title>
            <meta charset="utf-8">
            <meta name="viewport" content="width=device-width, initial-scale=1">
            <style>
                body {
                    font-family: Arial, sans-serif;
                    max-width: 1000px;
                    margin: 0 auto;
                    padding: 20px;
                    line-height: 1.6;
                    color: #333;
                }
                h1, h2, h3 {
                    color: #2c3e50;
                    margin-top: 30px;
                }
                h1 {
                    border-bottom: 1px solid #eee;
                    padding-bottom: 10px;
                }
                .container {
                    display: flex;
                    flex-wrap: wrap;
                    gap: 20px;
                }
                .api-doc, .test-form {
                    flex: 1;
                    min-width: 300px;
                    border: 1px solid #ddd;
                    border-radius: 5px;
                    padding: 20px;
                    background-color: #f9f9f9;
                }
                table {
                    width: 100%;
                    border-collapse: collapse;
                    margin: 20px 0;
                }
                table, th, td {
                    border: 1px solid #ddd;
                }
                th, td {
                    padding: 12px;
                    text-align: left;
                }
                th {
                    background-color: #f2f2f2;
                }
                pre, code {
                    background-color: #f5f5f5;
                    padding: 5px;
                    border-radius: 3px;
                    overflow-x: auto;
                    font-family: monospace;
                }
                pre {
                    padding: 10px;
                }
                .btn {
                    display: inline-block;
                    background: #3498db;
                    color: white;
                    padding: 10px 20px;
                    text-decoration: none;
                    border-radius: 4px;
                    border: none;
                    cursor: pointer;
                    font-size: 16px;
                }
                .btn:hover {
                    background: #2980b9;
                }
                input, select {
                    padding: 8px;
                    margin: 5px 0;
                    width: 100%;
                    border: 1px solid #ddd;
                    border-radius: 4px;
                }
                #result {
                    display: none;
                    margin-top: 20px;
                    padding: 15px;
                    border: 1px solid #ddd;
                    border-radius: 4px;
                    background-color: #f8f8f8;
                }
                .success {
                    color: #2ecc71;
                }
                .error {
                    color: #e74c3c;
                }
            </style>
        </head>
        <body>
            <h1>LibreOffice文档转换API</h1>
            <p>这是一个基于Go实现的文档转换API服务，可以将各种格式的文档转换为其他格式。</p>
            
            <div class="container">
                <div class="api-doc">
                    <h2>API文档</h2>
                    
                    <h3>1. 文档转换 API</h3>
                    <p><strong>接口</strong>: <code>POST /convert</code></p>
                    <p><strong>说明</strong>: 将上传的文档转换为指定格式</p>
                    <p><strong>请求参数</strong>:</p>
                    <table>
                        <tr>
                            <th>参数名</th>
                            <th>类型</th>
                            <th>必填</th>
                            <th>说明</th>
                        </tr>
                        <tr>
                            <td>file</td>
                            <td>File</td>
                            <td>是</td>
                            <td>要转换的文档文件</td>
                        </tr>
                        <tr>
                            <td>format</td>
                            <td>String</td>
                            <td>否</td>
                            <td>目标格式，默认为txt</td>
                        </tr>
                    </table>
                    
                    <p><strong>支持的格式</strong>:</p>
                    <ul>
                        <li>文本格式: <code>txt</code></li>
                        <li>PDF格式: <code>pdf</code></li>
                        <li>Word格式: <code>docx</code></li>
                        <li>Excel格式: <code>xlsx</code></li>
                        <li>HTML格式: <code>html</code></li>
                        <li>更多格式参考LibreOffice文档</li>
                    </ul>
                    
                    <p><strong>注意事项</strong>:</p>
                    <ul>
                        <li>并非所有格式都可以互相转换，转换能力取决于LibreOffice的支持情况</li>
                        <li>PDF转Word等复杂转换可能无法保留原始格式</li>
                        <li>转换失败时会返回详细的错误信息</li>
                        <li>大文件转换可能需要较长时间</li>
                        <li>建议在转换前备份原始文件</li>
                    </ul>
                    
                    <p><strong>响应示例</strong>:</p>
                    <pre>{
  "success": true,
  "filename": "原始文件名.docx",
  "download_url": "http://localhost:15000/download/20231201/文件名_1701410000000.pdf",
  "download_filename": "20231201/文件名_1701410000000.pdf",
  "expiry": "2023-12-02 10:00:00"
}</pre>
                    
                    <h3>2. 文件下载 API</h3>
                    <p><strong>接口</strong>: <code>GET /download/:filename</code></p>
                    <p><strong>说明</strong>: 下载已转换的文件</p>
                    <p><strong>请求参数</strong>: </p>
                    <table>
                        <tr>
                            <th>参数名</th>
                            <th>类型</th>
                            <th>必填</th>
                            <th>说明</th>
                        </tr>
                        <tr>
                            <td>filename</td>
                            <td>String</td>
                            <td>是</td>
                            <td>文件路径，格式为 日期/文件名，如：20231201/example_123456789.pdf</td>
                        </tr>
                    </table>
                    
                    <h3>3. 健康检查 API</h3>
                    <p><strong>接口</strong>: <code>GET /health</code></p>
                    <p><strong>说明</strong>: 检查服务健康状态</p>
                    <p><strong>响应示例</strong>:</p>
                    <pre>{
  "status": "healthy",
  "libreoffice": true,
  "version": "LibreOffice 7.5.3",
  "data_dir": "/app/data",
  "file_expiry_hours": 24
}</pre>
                </div>
                
                <div class="test-form">
                    <h2>转换测试</h2>
                    <form id="convertForm" enctype="multipart/form-data">
                        <div>
                            <label for="file">选择文件:</label>
                            <input type="file" id="file" name="file" required>
                        </div>
                        <div>
                            <label for="format">转换格式:</label>
                            <select id="format" name="format">
                                <option value="txt">文本 (txt)</option>
                                <option value="pdf">PDF (pdf)</option>
                                <option value="docx">Word 文档 (docx)</option>
                                <option value="doc">Word 97-2003 文档 (doc)</option>
                                <option value="odt">OpenDocument 文本 (odt)</option>
                                <option value="rtf">富文本格式 (rtf)</option>
                                <option value="xlsx">Excel 表格 (xlsx)</option>
                                <option value="xls">Excel 97-2003 表格 (xls)</option>
                                <option value="ods">OpenDocument 表格 (ods)</option>
                                <option value="csv">CSV 表格 (csv)</option>
                                <option value="pptx">PowerPoint 演示文稿 (pptx)</option>
                                <option value="ppt">PowerPoint 97-2003 演示文稿 (ppt)</option>
                                <option value="odp">OpenDocument 演示文稿 (odp)</option>
                                <option value="html">HTML 网页 (html)</option>
                            </select>
                        </div>
                        <div style="margin-top: 20px;">
                            <button type="submit" class="btn">开始转换</button>
                        </div>
                    </form>
                    
                    <div id="result">
                        <h3>转换结果</h3>
                        <div id="resultContent"></div>
                    </div>
                    
                    <script>
                        document.getElementById('convertForm').addEventListener('submit', function(e) {
                            e.preventDefault();
                            
                            var formData = new FormData();
                            var fileInput = document.getElementById('file');
                            var formatInput = document.getElementById('format');
                            
                            if (fileInput.files.length === 0) {
                                alert('请选择要转换的文件');
                                return;
                            }
                            
                            formData.append('file', fileInput.files[0]);
                            formData.append('format', formatInput.value);
                            
                            var resultDiv = document.getElementById('result');
                            var resultContent = document.getElementById('resultContent');
                            resultContent.innerHTML = '<p>正在转换，请稍候...</p>';
                            resultDiv.style.display = 'block';
                            
                            fetch('/convert', {
                                method: 'POST',
                                body: formData
                            })
                            .then(function(response) {
                                return response.json();
                            })
                            .then(function(data) {
                                if (data.success) {
                                    var html = '<div class="success">' +
                                        '<p>转换成功!</p>' +
                                        '<p>原始文件: ' + data.filename + '</p>' +
                                        '<p>格式: ' + formatInput.value + '</p>';
                                    
                                    if (data.expiry) {
                                        html += '<p>过期时间: ' + data.expiry + '</p>';
                                    }
                                    
                                    html += '<p><a href="' + data.download_url + '" target="_blank" class="btn">下载文件</a></p>';
                                    
                                    if (data.text) {
                                        html += '<h4>文件内容预览:</h4>' +
                                            '<pre style="max-height: 300px; overflow: auto;">' + data.text + '</pre>';
                                    }
                                    
                                    html += '</div>';
                                    resultContent.innerHTML = html;
                                } else {
                                    resultContent.innerHTML = '<div class="error">' +
                                        '<p>转换失败: ' + data.error + '</p>' +
                                        (data.details ? '<p>详情: ' + data.details + '</p>' : '') +
                                        '</div>';
                                }
                            })
                            .catch(function(error) {
                                resultContent.innerHTML = '<div class="error">' +
                                    '<p>请求失败: ' + error.message + '</p>' +
                                    '</div>';
                            });
                        });
                    </script>
                </div>
            </div>
            
            <footer style="margin-top: 50px; border-top: 1px solid #eee; padding-top: 20px; text-align: center; color: #777;">
                <p>LibreOffice文档转换API | Go版本 | 支持多种文档格式转换</p>
            </footer>
        </body>
    </html>
    `
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// 健康检查处理
func healthCheckHandler(c *gin.Context) {
	response := HealthResponse{
		Status:         "healthy",
		LibreOffice:    libreofficeAvailable,
		Version:        libreofficeVersion,
		DataDir:        DATA_DIR,
		FileExpiryHours: FILE_EXPIRY_HOURS,
	}
	c.JSON(http.StatusOK, response)
}

// 文件下载处理
func downloadFileHandler(c *gin.Context) {
	filename := c.Param("filename")
	parts := strings.Split(filename, "/")
	
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文件路径"})
		return
	}
	
	dateDir, fileName := parts[0], parts[1]
	filePath := filepath.Join(DATA_DIR, dateDir, fileName)
	
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "文件不存在"})
		return
	}
	
	c.File(filePath)
}

// 文档转换处理
func convertDocumentHandler(c *gin.Context) {
	// 检查LibreOffice是否可用
	if !libreofficeAvailable {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "LibreOffice未安装或配置错误",
			Details: libreofficeVersion,
		})
		return
	}
	
	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "没有上传文件"})
		return
	}
	defer file.Close()
	
	if header.Filename == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "文件名为空"})
		return
	}
	
	// 获取转换格式，默认为txt
	convertFormat := c.PostForm("format")
	if convertFormat == "" {
		convertFormat = "txt"
	}
	
	// 从格式中提取扩展名
	targetExt := strings.ToLower(strings.Split(convertFormat, ":")[0])
	
	log.Printf("使用转换格式: %s, 目标扩展名: %s", convertFormat, targetExt)
	
	// 获取原始文件名和扩展名
	originalFilename := header.Filename
	fileExt := strings.ToLower(filepath.Ext(originalFilename))
	
	// 使用唯一ID作为文件名，避免中文文件名问题
	uniqueID := uuid.New().String()
	safeFilename := fmt.Sprintf("%s%s", uniqueID, fileExt)
	
	log.Printf("接收到文件: %s, 使用安全文件名: %s", originalFilename, safeFilename)
	
	// 在tmp目录下创建一个新的子目录用于此次转换
	workDir := filepath.Join(TMP_DIR, fmt.Sprintf("work_%s", uniqueID))
	os.MkdirAll(workDir, 0755)
	
	// 保存上传的文件
	filePath := filepath.Join(workDir, safeFilename)
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("创建文件失败: %v", err)})
		return
	}
	
	// 复制文件内容
	if _, err = io.Copy(dst, file); err != nil {
		dst.Close()
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("保存文件失败: %v", err)})
		return
	}
	dst.Close()
	
	// 转换文件并响应
	response, statusCode := convertFile(workDir, filePath, originalFilename, convertFormat, targetExt, uniqueID, c)
	
	// 清理临时目录
	defer func() {
		if err := os.RemoveAll(workDir); err != nil {
			log.Printf("清理临时目录时出错: %v", err)
		} else {
			log.Printf("清理临时目录: %s", workDir)
		}
	}()
	
	c.JSON(statusCode, response)
}

// 文件转换处理
func convertFile(workDir, filePath, originalFilename, convertFormat, targetExt, uniqueID string, c *gin.Context) (interface{}, int) {
	// 检查是否需要通过中间格式转换
	needsIntermediateFormat := false
	intermediateFormat := ""
	fileExt := strings.ToLower(filepath.Ext(filePath))
	
	// 特殊情况处理：PDF到TXT需要通过ODT中间格式
	if fileExt == ".pdf" && targetExt == "txt" {
		needsIntermediateFormat = true
		intermediateFormat = "odt"
		log.Printf("检测到PDF到TXT的转换，将使用中间格式ODT")
	}
	
	var outputPath string
	
	// 如果需要中间转换
	if needsIntermediateFormat {
		// 第一步：转换为中间格式
		intermediateCmd := []string{
			"--headless",
			"--convert-to",
			intermediateFormat,
			filePath,
			"--outdir",
			workDir,
		}
		
		log.Printf("执行中间转换命令: %s %s", SOFFICE_PATH, strings.Join(intermediateCmd, " "))
		
		// 执行中间转换命令
		cmd := exec.Command(SOFFICE_PATH, intermediateCmd...)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		log.Printf("中间转换命令输出: %s", outputStr)
		
		// 检查命令是否出错
		if err != nil {
			log.Printf("中间转换过程出错: %v, 输出: %s", err, outputStr)
			return ErrorResponse{
				Error:   "中间转换失败",
				Details: fmt.Sprintf("%v: %s", err, outputStr),
			}, http.StatusInternalServerError
		}
		
		// 检查中间转换输出是否包含错误信息
		if strings.Contains(outputStr, "Error:") || strings.Contains(outputStr, "error") || 
		   strings.Contains(outputStr, "Failed") || strings.Contains(outputStr, "failed") ||
		   strings.Contains(outputStr, "no export filter") {
			log.Printf("中间转换过程有错误信息: %s", outputStr)
			return ErrorResponse{
				Error:   "中间转换失败",
				Details: fmt.Sprintf("LibreOffice报告错误: %s", outputStr),
			}, http.StatusInternalServerError
		}
		
		// 找到中间格式文件
		var intermediateFilePath string
		files, err := os.ReadDir(workDir)
		if err != nil {
			return ErrorResponse{Error: fmt.Sprintf("读取工作目录失败: %v", err)}, http.StatusInternalServerError
		}
		
		for _, f := range files {
			fname := f.Name()
			if strings.HasSuffix(strings.ToLower(fname), fmt.Sprintf(".%s", intermediateFormat)) &&
			   fname != filepath.Base(filePath) {
				intermediateFilePath = filepath.Join(workDir, fname)
				log.Printf("找到中间格式文件: %s", intermediateFilePath)
				break
			}
		}
		
		if intermediateFilePath == "" {
			log.Printf("未找到中间格式文件")
			return ErrorResponse{
				Error:   "中间转换失败",
				Details: "未找到中间格式文件",
			}, http.StatusInternalServerError
		}
		
		// 第二步：将中间格式转换为目标格式
		finalCmd := []string{
			"--headless",
			"--convert-to",
			convertFormat,
			intermediateFilePath,
			"--outdir",
			workDir,
		}
		
		log.Printf("执行最终转换命令: %s %s", SOFFICE_PATH, strings.Join(finalCmd, " "))
		
		// 执行最终转换命令
		cmd = exec.Command(SOFFICE_PATH, finalCmd...)
		output, err = cmd.CombinedOutput()
		outputStr = string(output)
		log.Printf("最终转换命令输出: %s", outputStr)
		
		// 检查命令是否出错
		if err != nil {
			log.Printf("最终转换过程出错: %v, 输出: %s", err, outputStr)
			return ErrorResponse{
				Error:   "最终转换失败",
				Details: fmt.Sprintf("%v: %s", err, outputStr),
			}, http.StatusInternalServerError
		}
		
		// 检查最终转换输出是否包含错误信息
		if strings.Contains(outputStr, "Error:") || strings.Contains(outputStr, "error") || 
		   strings.Contains(outputStr, "Failed") || strings.Contains(outputStr, "failed") ||
		   strings.Contains(outputStr, "no export filter") {
			log.Printf("最终转换过程有错误信息: %s", outputStr)
			return ErrorResponse{
				Error:   "最终转换失败",
				Details: fmt.Sprintf("LibreOffice报告错误: %s", outputStr),
			}, http.StatusInternalServerError
		}
	} else {
		// 使用LibreOffice将文件转换为指定格式
		convertCmd := []string{
			"--headless",
			"--convert-to",
			convertFormat,
			filePath,
			"--outdir",
			workDir,
		}
		
		log.Printf("执行转换命令: %s %s", SOFFICE_PATH, strings.Join(convertCmd, " "))
		
		// 执行转换命令
		cmd := exec.Command(SOFFICE_PATH, convertCmd...)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		log.Printf("命令输出: %s", outputStr)
		
		// 检查命令是否出错
		if err != nil {
			log.Printf("转换过程出错: %v, 输出: %s", err, outputStr)
			return ErrorResponse{
				Error:   "文件转换失败",
				Details: fmt.Sprintf("%v: %s", err, outputStr),
			}, http.StatusInternalServerError
		}
		
		// 即使命令返回成功，也检查输出中是否包含错误信息
		if strings.Contains(outputStr, "Error:") || strings.Contains(outputStr, "error") || 
		   strings.Contains(outputStr, "Failed") || strings.Contains(outputStr, "failed") ||
		   strings.Contains(outputStr, "no export filter") {
			log.Printf("转换过程有错误信息: %s", outputStr)
			return ErrorResponse{
				Error:   "文件转换失败",
				Details: fmt.Sprintf("LibreOffice报告错误: %s", outputStr),
			}, http.StatusInternalServerError
		}
	}
	
	// 列出目录中的所有文件
	files, err := os.ReadDir(workDir)
	if err != nil {
		return ErrorResponse{Error: fmt.Sprintf("读取工作目录失败: %v", err)}, http.StatusInternalServerError
	}
	
	var allFiles []string
	for _, f := range files {
		allFiles = append(allFiles, f.Name())
	}
	log.Printf("工作目录中的所有文件: %v", allFiles)
	
	// 获取输入文件的基本名称（不含路径和扩展名）
	baseInputFilename := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	log.Printf("基本输入文件名: %s", baseInputFilename)
	
	// 尝试查找转换后的文件
	outputPath = ""
	
	// 查找目标扩展名的文件，但不是原始输入文件
	for _, file := range allFiles {
		// 检查文件是否有目标扩展名
		if strings.HasSuffix(strings.ToLower(file), fmt.Sprintf(".%s", targetExt)) {
			// 检查这不是原始输入文件（如果扩展名相同）
			if file != filepath.Base(filePath) {
				outputPath = filepath.Join(workDir, file)
				log.Printf("找到输出文件: %s", outputPath)
				break
			}
		}
	}
	
	// 如果仍然没有找到输出文件，则报错
	if outputPath == "" {
		// 提供更具体的错误信息
		errorDetails := fmt.Sprintf("在工作目录中未找到以 .%s 结尾的转换输出文件。这可能由于以下原因：\n", targetExt)
		errorDetails += "1. 当前文件格式不支持转换到请求的目标格式\n"
		errorDetails += "2. 源文件可能已损坏或格式不兼容\n"
		errorDetails += "3. LibreOffice未能正确执行转换过程\n"
		
		return ErrorResponse{
			Error:   "转换后的文件未找到",
			Details: errorDetails + fmt.Sprintf("工作目录中的文件: %v", allFiles),
		}, http.StatusInternalServerError
	}
	
	log.Printf("转换后的输出文件路径: %s", outputPath)
	
	// 生成持久化存储路径
	finalOutputPath, relativePath := generateOutputFilepath(originalFilename, targetExt)
	
	// 将文件从临时目录复制到持久化存储目录
	src, err := os.Open(outputPath)
	if err != nil {
		return ErrorResponse{Error: fmt.Sprintf("打开输出文件失败: %v", err)}, http.StatusInternalServerError
	}
	defer src.Close()
	
	dst, err := os.Create(finalOutputPath)
	if err != nil {
		return ErrorResponse{Error: fmt.Sprintf("创建持久化文件失败: %v", err)}, http.StatusInternalServerError
	}
	defer dst.Close()
	
	if _, err = io.Copy(dst, src); err != nil {
		return ErrorResponse{Error: fmt.Sprintf("复制文件失败: %v", err)}, http.StatusInternalServerError
	}
	
	log.Printf("已将文件复制到持久化存储目录: %s", finalOutputPath)
	
	// 生成下载URL
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	downloadURL := fmt.Sprintf("%s://%s/download/%s", scheme, host, relativePath)
	
	// 计算过期时间
	var expiryInfo string
	if FILE_EXPIRY_HOURS > 0 {
		expiryTime := time.Now().Add(time.Duration(FILE_EXPIRY_HOURS) * time.Hour)
		expiryInfo = expiryTime.Format("2006-01-02 15:04:05")
	} else {
		expiryInfo = "永不过期"
	}
	
	// 构建响应
	response := ConversionResponse{
		Success:         true,
		Filename:        originalFilename,
		DownloadURL:     downloadURL,
		DownloadFilename: relativePath,
		Expiry:          expiryInfo,
	}
	
	// 如果输出是文本格式，则读取文本内容
	if targetExt == "txt" {
		textBytes, err := os.ReadFile(finalOutputPath)
		if err == nil {
			response.Text = string(textBytes)
		} else {
			log.Printf("读取文本内容失败: %v", err)
		}
	}
	
	return response, http.StatusOK
} 