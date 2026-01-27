// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package sshclient

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client SSH客户端
type Client struct {
	client *ssh.Client
}

// NewClient 创建SSH客户端
func NewClient(host string, port int, username, password string, privateKey []byte, passphrase string) (*Client, error) {
	var authMethods []ssh.AuthMethod

	// 优先使用私钥认证
	if len(privateKey) > 0 {
		var signer ssh.Signer
		var err error

		if passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(privateKey, []byte(passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(privateKey)
		}

		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// 密码认证
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("至少需要一种认证方式")
	}

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证主机密钥
		Timeout:         10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %w", err)
	}

	return &Client{client: client}, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Execute 执行命令
func (c *Client) Execute(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建session失败: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	if err != nil {
		return "", fmt.Errorf("命令执行失败: %s, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// ExecuteWithTimeout 执行命令（带超时）
func (c *Client) ExecuteWithTimeout(command string, timeout time.Duration) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建session失败: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// 使用通道实现超时
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case <-time.After(timeout):
		// 超时，尝试关闭session
		session.Signal(ssh.SIGTERM)
		session.Close()
		return "", fmt.Errorf("命令执行超时")
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("命令执行失败: %s, stderr: %s", err, stderr.String())
		}
	}

	return stdout.String(), nil
}

// TestConnection 测试连接
func (c *Client) TestConnection() error {
	_, err := c.Execute("echo ok")
	return err
}

// NewSFTPClient 创建SFTP客户端
func (c *Client) NewSFTPClient() (*sftp.Client, error) {
	return sftp.NewClient(c.client)
}

// FileInfo 文件信息
type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
}

// ListDir 列出目录内容
func (c *Client) ListDir(remotePath string) ([]*FileInfo, error) {
	sftpClient, err := c.NewSFTPClient()
	if err != nil {
		return nil, fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	files, err := sftpClient.ReadDir(remotePath)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	var fileList []*FileInfo
	for _, file := range files {
		fileInfo := &FileInfo{
			Name:    file.Name(),
			Size:    file.Size(),
			Mode:    file.Mode().String(),
			IsDir:   file.IsDir(),
			ModTime: file.ModTime().Format("2006-01-02 15:04:05"),
		}
		fileList = append(fileList, fileInfo)
	}

	return fileList, nil
}

// UploadFile 上传文件
func (c *Client) UploadFile(localPath, remotePath string) error {
	sftpClient, err := c.NewSFTPClient()
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	// 打开本地文件
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer localFile.Close()

	// 创建远程文件
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	// 复制文件内容
	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}

	return nil
}

// UploadFromReader 从 Reader 上传文件
func (c *Client) UploadFromReader(reader io.Reader, remotePath string) error {
	sftpClient, err := c.NewSFTPClient()
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	// 创建远程文件
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	// 复制文件内容
	_, err = io.Copy(remoteFile, reader)
	if err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}

	return nil
}

// DownloadFile 下载文件
func (c *Client) DownloadFile(remotePath, localPath string) error {
	sftpClient, err := c.NewSFTPClient()
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	// 打开远程文件
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("打开远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	// 创建本地文件
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %w", err)
	}
	defer localFile.Close()

	// 复制文件内容
	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}

	return nil
}

// DownloadToWriter 下载文件到 Writer
func (c *Client) DownloadToWriter(remotePath string, writer io.Writer) error {
	sftpClient, err := c.NewSFTPClient()
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	// 打开远程文件
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("打开远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	// 复制文件内容
	_, err = io.Copy(writer, remoteFile)
	if err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}

	return nil
}

// RemoveFile 删除文件
func (c *Client) RemoveFile(remotePath string) error {
	sftpClient, err := c.NewSFTPClient()
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	err = sftpClient.Remove(remotePath)
	if err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}

	return nil
}

// MkDir 创建目录
func (c *Client) MkDir(remotePath string) error {
	sftpClient, err := c.NewSFTPClient()
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	err = sftpClient.Mkdir(remotePath)
	if err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	return nil
}

// MkdirAll 递归创建目录
func (c *Client) MkdirAll(remotePath string) error {
	sftpClient, err := c.NewSFTPClient()
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	err = sftpClient.MkdirAll(remotePath)
	if err != nil {
		return fmt.Errorf("递归创建目录失败: %w", err)
	}

	return nil
}

// StatFile 获取文件信息
func (c *Client) StatFile(remotePath string) (*FileInfo, error) {
	sftpClient, err := c.NewSFTPClient()
	if err != nil {
		return nil, fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	fileInfo, err := sftpClient.Stat(remotePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return &FileInfo{
		Name:    filepath.Base(remotePath),
		Size:    fileInfo.Size(),
		Mode:    fileInfo.Mode().String(),
		IsDir:   fileInfo.IsDir(),
		ModTime: fileInfo.ModTime().Format("2006-01-02 15:04:05"),
	}, nil
}
