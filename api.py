import os
import tempfile
import subprocess
import shutil
from flask import Flask, request, jsonify, send_from_directory, url_for
import logging
from flasgger import Swagger, swag_from
from werkzeug.utils import secure_filename
import uuid
import time
import datetime
import threading
import glob

# 加载.env
from dotenv import load_dotenv
load_dotenv()

# 从环境变量获取配置
DEBUG = os.environ.get('DEBUG', 'True').lower() == 'true'
MAX_CONTENT_LENGTH = int(os.environ.get('MAX_CONTENT_LENGTH', 100 * 1024 * 1024))  # 默认100MB
SOFFICE_PATH = os.environ.get('SOFFICE_PATH', 'soffice')
FILE_EXPIRY_HOURS = os.environ.get('FILE_EXPIRY_HOURS', '24')  # 文件过期时间（小时）

# 如果FILE_EXPIRY_HOURS是空字符串或-1，设置为None表示永不过期
if FILE_EXPIRY_HOURS == '' or FILE_EXPIRY_HOURS == '-1':
    FILE_EXPIRY_HOURS = None
else:
    FILE_EXPIRY_HOURS = int(FILE_EXPIRY_HOURS)

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
TMP_DIR = os.path.join(BASE_DIR, 'tmp')
DATA_DIR = os.path.join(BASE_DIR, 'data')

# 创建必要的目录
os.makedirs(TMP_DIR, exist_ok=True)
os.makedirs(DATA_DIR, exist_ok=True)

app = Flask(__name__)
app.config['MAX_CONTENT_LENGTH'] = MAX_CONTENT_LENGTH

# 配置Swagger
swagger_config = {
    "headers": [],
    "specs": [
        {
            "endpoint": "apispec",
            "route": "/apispec.json",
            "rule_filter": lambda rule: True,
            "model_filter": lambda tag: True,
        }
    ],
    "static_url_path": "/flasgger_static",
    "swagger_ui": True,
    "specs_route": "/docs/"
}

swagger_template = {
    "swagger": "2.0",
    "info": {
        "title": "LibreOffice文档转换API",
        "description": "将各种格式的文档转换为指定格式",
        "version": "1.0.0",
        "contact": {
            "name": "API Support",
            "email": "support@example.com"
        }
    }
}

swagger = Swagger(app, config=swagger_config, template=swagger_template)

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# 设置LibreOffice可执行文件路径
soffice_path = SOFFICE_PATH  # 从环境变量获取

# 清理过期文件的函数
def cleanup_expired_files():
    """清理过期的文件"""
    if FILE_EXPIRY_HOURS is None:
        logger.info("文件永不过期，跳过清理")
        return
    
    now = time.time()
    expiry_seconds = FILE_EXPIRY_HOURS * 3600
    
    # 搜索DATA_DIR下所有子目录
    for root, dirs, files in os.walk(DATA_DIR):
        for file in files:
            file_path = os.path.join(root, file)
            # 获取文件创建时间
            file_creation_time = os.path.getctime(file_path)
            # 如果文件超过过期时间，则删除
            if now - file_creation_time > expiry_seconds:
                try:
                    os.remove(file_path)
                    logger.info(f"已删除过期文件: {file_path}")
                except Exception as e:
                    logger.error(f"删除过期文件时出错: {e}")
    
    # 删除空目录
    for root, dirs, files in os.walk(DATA_DIR, topdown=False):
        for dir in dirs:
            dir_path = os.path.join(root, dir)
            if not os.listdir(dir_path):  # 如果目录为空
                try:
                    os.rmdir(dir_path)
                    logger.info(f"已删除空目录: {dir_path}")
                except Exception as e:
                    logger.error(f"删除空目录时出错: {e}")

# 启动定时清理任务
def start_cleanup_scheduler():
    """启动定时清理任务"""
    if FILE_EXPIRY_HOURS is None:
        logger.info("文件永不过期，不启动清理任务")
        return
    
    def run_cleanup():
        while True:
            logger.info("开始执行文件清理任务")
            cleanup_expired_files()
            # 每小时检查一次
            time.sleep(3600)
    
    # 启动清理线程
    cleanup_thread = threading.Thread(target=run_cleanup, daemon=True)
    cleanup_thread.start()
    logger.info(f"已启动文件清理任务，过期时间: {FILE_EXPIRY_HOURS}小时")

# 检查LibreOffice是否可用
def check_libreoffice():
    try:
        result = subprocess.run(
            [soffice_path, '--version'],
            check=False,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=10
        )
        if result.returncode == 0:
            version = result.stdout.decode('utf-8', errors='ignore').strip()
            logger.info(f"检测到LibreOffice: {version}")
            return True, version
        else:
            logger.error(f"LibreOffice检测失败: {result.stderr.decode('utf-8', errors='ignore')}")
            return False, result.stderr.decode('utf-8', errors='ignore')
    except Exception as e:
        logger.error(f"LibreOffice检测异常: {e}")
        return False, str(e)

# 尝试加载LibreOffice
libreoffice_available, libreoffice_version = check_libreoffice()

# 生成持久化文件的存储路径
def generate_output_filepath(original_filename, target_ext):
    """生成输出文件的路径，格式为DATA_DIR/yyyymmdd/[input_filename]_[suffix].xxx"""
    # 获取当前日期
    date_str = datetime.datetime.now().strftime('%Y%m%d')
    date_dir = os.path.join(DATA_DIR, date_str)
    
    # 确保日期目录存在
    os.makedirs(date_dir, exist_ok=True)
    
    # 提取原始文件名（不含扩展名）
    base_name = os.path.splitext(original_filename)[0]
    # 生成时间戳后缀
    timestamp_suffix = int(time.time() * 1000)
    
    # 构建输出文件名
    output_filename = f"{base_name}_{timestamp_suffix}.{target_ext}"
    output_filepath = os.path.join(date_dir, output_filename)
    
    # 相对路径（用于构建URL）
    relative_path = os.path.join(date_str, output_filename)
    
    return output_filepath, relative_path

@app.route('/download/<path:filename>', methods=['GET'])
@swag_from({
    "tags": ["文件下载"],
    "summary": "下载已转换的文件",
    "description": "根据文件路径下载已转换的文件",
    "parameters": [
        {
            "name": "filename",
            "in": "path",
            "type": "string",
            "required": True,
            "description": "要下载的文件路径"
        }
    ],
    "responses": {
        200: {
            "description": "文件下载成功"
        },
        404: {
            "description": "文件不存在",
            "schema": {
                "type": "object",
                "properties": {
                    "error": {
                        "type": "string",
                        "example": "文件不存在"
                    }
                }
            }
        }
    }
})
def download_file(filename):
    """下载转换后的文件"""
    # 分割路径，获取日期目录和文件名
    parts = filename.split('/')
    if len(parts) != 2:
        return jsonify({"error": "无效的文件路径"}), 400
    
    date_dir, file_name = parts
    
    return send_from_directory(os.path.join(DATA_DIR, date_dir), file_name)

@app.route('/convert', methods=['POST'])
@swag_from({
    "tags": ["文档转换"],
    "summary": "将文档转换为指定格式",
    "description": "接收上传的文档文件，使用LibreOffice将其转换为指定格式并返回下载链接",
    "parameters": [
        {
            "name": "file",
            "in": "formData",
            "type": "file",
            "required": True,
            "description": "要转换的文档文件（支持.docx, .xlsx, .pptx, .pdf等格式）"
        },
        {
            "name": "format",
            "in": "formData",
            "type": "string",
            "required": False,
            "description": "转换的目标格式，默认为txt。可选值包括pdf, docx, doc, xlsx等"
        }
    ],
    "consumes": ["multipart/form-data"],
    "produces": ["application/json"],
    "responses": {
        200: {
            "description": "成功转换文档",
            "schema": {
                "type": "object",
                "properties": {
                    "success": {
                        "type": "boolean",
                        "example": True
                    },
                    "filename": {
                        "type": "string",
                        "example": "document.docx"
                    },
                    "download_url": {
                        "type": "string",
                        "example": "/download/20231201/document_1701410000000.pdf"
                    },
                    "download_filename": {
                        "type": "string",
                        "example": "20231201/document_1701410000000.pdf"
                    },
                    "text": {
                        "type": "string",
                        "example": "这是文档中的文本内容..."
                    },
                    "expiry": {
                        "type": "string",
                        "example": "2023-12-02 10:00:00"
                    }
                }
            }
        },
        400: {
            "description": "请求无效",
            "schema": {
                "type": "object",
                "properties": {
                    "error": {
                        "type": "string",
                        "example": "没有上传文件"
                    }
                }
            }
        },
        500: {
            "description": "服务器错误",
            "schema": {
                "type": "object",
                "properties": {
                    "error": {
                        "type": "string",
                        "example": "文件转换失败"
                    },
                    "details": {
                        "type": "string",
                        "example": "转换过程中出现错误"
                    }
                }
            }
        }
    }
})
def convert_document():
    """
    接收文件上传并转换为指定格式
    请求参数: 
        - file: 要转换的文件（multipart/form-data）
        - format: 转换的目标格式，默认为txt
    返回:
        - 转换后的文件下载链接和内容（如果是文本格式）
    """
    if not libreoffice_available:
        return jsonify({
            "error": "LibreOffice未安装或配置错误",
            "details": libreoffice_version
        }), 500
        
    if 'file' not in request.files:
        return jsonify({"error": "没有上传文件"}), 400
    
    file = request.files['file']
    if file.filename == '':
        return jsonify({"error": "文件名为空"}), 400
    
    # 获取转换格式，默认为txt
    convert_format = request.form.get('format', 'txt')
    # 从格式中提取扩展名
    target_ext = convert_format.split(':')[0].lower()
    
    logger.info(f"使用转换格式: {convert_format}, 目标扩展名: {target_ext}")
    
    try:
        # 获取原始文件名和扩展名
        original_filename = file.filename
        file_ext = os.path.splitext(original_filename)[1].lower()
        
        # 使用唯一ID作为文件名，避免中文文件名问题
        unique_id = str(uuid.uuid4())
        safe_filename = f"{unique_id}{file_ext}"
        
        logger.info(f"接收到文件: {original_filename}, 使用安全文件名: {safe_filename}")
        
        # 在tmp目录下创建一个新的子目录用于此次转换
        work_dir = os.path.join(TMP_DIR, f"work_{unique_id}")
        os.makedirs(work_dir, exist_ok=True)
        
        try:
            # 保存文件
            file_path = os.path.join(work_dir, safe_filename)
            file.save(file_path)
            
            logger.info(f"文件保存至: {file_path}")
            
            # 使用LibreOffice将文件转换为指定格式
            convert_cmd = [
                soffice_path, 
                '--headless', 
                '--convert-to', 
                convert_format,
                file_path, 
                '--outdir', 
                work_dir
            ]
            
            logger.info(f"执行转换命令: {' '.join(convert_cmd)}")
            
            # 执行转换并捕获所有输出
            result = subprocess.run(
                convert_cmd,
                check=False,  # 不自动抛出异常，我们自己处理结果
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                timeout=60  # 增加超时时间
            )
            
            # 打印命令输出以便调试
            stdout = result.stdout.decode('utf-8', errors='ignore')
            stderr = result.stderr.decode('utf-8', errors='ignore')
            logger.info(f"命令输出: {stdout}")
            
            if stderr:
                logger.warning(f"命令错误输出: {stderr}")
            
            if result.returncode != 0:
                raise subprocess.CalledProcessError(
                    result.returncode, 
                    convert_cmd, 
                    output=result.stdout, 
                    stderr=result.stderr
                )
            
            # 列出目录中的所有文件
            all_files = os.listdir(work_dir)
            logger.info(f"工作目录中的所有文件: {all_files}")
            
            # 获取预期的转换后文件名
            expected_output_filename = f"{unique_id}.{target_ext}"
            expected_output_path = os.path.join(work_dir, expected_output_filename)
            
            # 尝试查找转换后的文件
            output_path = None
            
            # 1. 尝试直接查找预期文件名
            if os.path.exists(expected_output_path):
                output_path = expected_output_path
                logger.info(f"找到预期的输出文件: {output_path}")
            
            # 2. 如果没找到，尝试查找任何与目标扩展名匹配的文件
            if output_path is None:
                for filename in all_files:
                    if filename.lower().endswith(f'.{target_ext}'):
                        output_path = os.path.join(work_dir, filename)
                        logger.info(f"找到其他输出文件: {output_path}")
                        break
            
            # 3. 如果是TXT格式且仍然找不到，尝试使用备用方法提取文本
            if target_ext == 'txt' and output_path is None and os.path.exists(file_path):
                # 如果是PDF，尝试使用其他工具提取文本
                if file_ext.lower() == '.pdf':
                    try:
                        import PyPDF2
                        txt_content = ""
                        with open(file_path, 'rb') as pdf_file:
                            pdf_reader = PyPDF2.PdfReader(pdf_file)
                            for page_num in range(len(pdf_reader.pages)):
                                page = pdf_reader.pages[page_num]
                                txt_content += page.extract_text() + "\n\n"
                        
                        # 创建文本文件
                        output_path = os.path.join(work_dir, expected_output_filename)
                        with open(output_path, 'w', encoding='utf-8') as txt_file:
                            txt_file.write(txt_content)
                        
                        logger.info(f"使用PyPDF2成功提取PDF文本: {output_path}")
                    except Exception as e:
                        logger.error(f"使用PyPDF2提取PDF文本失败: {e}")
            
            # 如果仍然没有找到输出文件，则报错
            if output_path is None:
                raise FileNotFoundError(f"转换后的文件未找到，文件列表: {all_files}")
            
            logger.info(f"转换后的输出文件路径: {output_path}")
            
            # 生成持久化存储路径
            final_output_path, relative_path = generate_output_filepath(original_filename, target_ext)
            
            # 将文件从临时目录移动到持久化存储目录
            shutil.copy2(output_path, final_output_path)
            logger.info(f"已将文件复制到持久化存储目录: {final_output_path}")
            
            # 生成下载URL
            download_url = url_for('download_file', filename=relative_path, _external=True)
            
            # 计算过期时间
            expiry_info = None
            if FILE_EXPIRY_HOURS is not None:
                expiry_time = datetime.datetime.now() + datetime.timedelta(hours=FILE_EXPIRY_HOURS)
                expiry_info = expiry_time.strftime('%Y-%m-%d %H:%M:%S')
            
            # 读取转换后的内容
            response_data = {
                "success": True,
                "filename": original_filename,
                "download_url": download_url,
                'download_filename': relative_path
            }
            
            # 添加过期时间信息
            if expiry_info:
                response_data["expiry"] = expiry_info
            else:
                response_data["expiry"] = "永不过期"
            
            # 如果输出是文本格式，则读取文本内容
            if target_ext == 'txt':
                with open(final_output_path, 'r', errors='ignore') as f:
                    text_content = f.read()
                response_data["text"] = text_content
            
            return jsonify(response_data)
            
        finally:
            # 清理临时目录
            try:
                if os.path.exists(work_dir):
                    shutil.rmtree(work_dir)
                    logger.info(f"清理临时目录: {work_dir}")
            except Exception as e:
                logger.warning(f"清理临时目录时出错: {e}")
    
    except subprocess.CalledProcessError as e:
        logger.error(f"转换过程出错: {e}")
        logger.error(f"标准输出: {e.stdout.decode('utf-8', errors='ignore')}")
        logger.error(f"标准错误: {e.stderr.decode('utf-8', errors='ignore')}")
        return jsonify({
            "error": "文件转换失败",
            "details": str(e),
            "stdout": e.stdout.decode('utf-8', errors='ignore'),
            "stderr": e.stderr.decode('utf-8', errors='ignore')
        }), 500
    except Exception as e:
        logger.error(f"处理过程出错: {e}", exc_info=True)
        return jsonify({"error": str(e)}), 500

@app.route('/health', methods=['GET'])
@swag_from({
    "tags": ["系统"],
    "summary": "健康检查接口",
    "description": "用于检查API服务是否正常运行",
    "responses": {
        200: {
            "description": "服务正常运行",
            "schema": {
                "type": "object",
                "properties": {
                    "status": {
                        "type": "string",
                        "example": "healthy"
                    },
                    "libreoffice": {
                        "type": "boolean",
                        "example": True
                    },
                    "version": {
                        "type": "string",
                        "example": "LibreOffice 7.5.3"
                    },
                    "data_dir": {
                        "type": "string",
                        "example": "/app/data"
                    },
                    "file_expiry_hours": {
                        "type": "integer",
                        "example": 24
                    }
                }
            }
        }
    }
})
def health_check():
    """健康检查接口"""
    return jsonify({
        "status": "healthy",
        "libreoffice": libreoffice_available,
        "version": libreoffice_version if libreoffice_available else "未安装",
        "data_dir": DATA_DIR,
        "file_expiry_hours": FILE_EXPIRY_HOURS
    })

@app.route('/', methods=['GET'])
def index():
    """重定向到API文档页面"""
    return """
    <html>
        <head>
            <title>LibreOffice文档转换API</title>
            <style>
                body {
                    font-family: Arial, sans-serif;
                    max-width: 800px;
                    margin: 0 auto;
                    padding: 20px;
                    line-height: 1.6;
                }
                h1 {
                    color: #2c3e50;
                    border-bottom: 1px solid #eee;
                    padding-bottom: 10px;
                }
                a {
                    display: inline-block;
                    background: #3498db;
                    color: white;
                    padding: 10px 20px;
                    text-decoration: none;
                    border-radius: 4px;
                    margin-top: 20px;
                }
                a:hover {
                    background: #2980b9;
                }
            </style>
        </head>
        <body>
            <h1>LibreOffice文档转换API</h1>
            <p>这是一个基于Flask实现的文档转换API服务，可以将各种格式的文档转换为其他格式。</p>
            <a href="/docs/">查看API文档</a>
        </body>
    </html>
    """

if __name__ == '__main__':
    # 启动文件清理任务
    start_cleanup_scheduler()
    
    logger.info(f"启动服务: host=0.0.0.0, port=15000, debug={DEBUG}")
    logger.info(f"文件存储目录: {DATA_DIR}, 过期时间: {FILE_EXPIRY_HOURS if FILE_EXPIRY_HOURS is not None else '永不过期'} 小时")
    app.run(debug=DEBUG, host='0.0.0.0', port=15000)
