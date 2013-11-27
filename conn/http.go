/*                                                                              
 * Copyright (C) 2013 Deepin, Inc.                                                 
 *               2013 Leslie Zhai <zhaixiang@linuxdeepin.com>                   
 *                                                                              
 * This program is free software: you can redistribute it and/or modify         
 * it under the terms of the GNU General Public License as published by         
 * the Free Software Foundation, either version 3 of the License, or            
 * any later version.                                                           
 *                                                                              
 * This program is distributed in the hope that it will be useful,              
 * but WITHOUT ANY WARRANTY; without even the implied warranty of               
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                
 * GNU General Public License for more details.                                 
 *                                                                              
 * You should have received a copy of the GNU General Public License            
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.        
 */

package conn

import (
    "fmt"
    "net"
    "regexp"
    "strings"
    "strconv"
    "os"
    "io"
)

type HTTP struct {
    Protocol        string
    host            string
    port            int
    user            string
    passwd          string
    Debug           bool
    UserAgent       string
    conn            net.Conn
    header          string
    headerResponse  string
    offset          int
    Error           error
    Callback        func(int)
}

const (
    buffer_size int = 102400
)

func (http *HTTP) Connect(host string, port int) {
    http.conn, http.Error = net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
    if http.Error != nil {
        fmt.Println("ERROR: ", http.Error.Error())
        return
    }
    http.host = host
    http.port = port
}

func (http *HTTP) AddHeader(header string) {
    http.header += header + "\r\n\r\n"
}

/* TODO: get http header */
func (http *HTTP) Response() {
    for i := 0; ; {
        data := make([]byte, 1)
        n, err := http.conn.Read(data)
        if err != nil {
            if err != io.EOF {
                fmt.Println("ERROR:", err.Error())
                defer http.conn.Close()
                http.Error = err
                return
            }
        }
        if data[0] == '\r' {
            continue
        } else if data[0] == '\n' {
            if i == 0 {
                break
            }
            i = 0
        } else {
            i++
        }
        http.headerResponse += string(data[:n])
    }
    if http.Debug {
        fmt.Println("DEBUG:", http.headerResponse)
    }
    http.conn.Close()
}

func (http *HTTP) WriteToFile(f *os.File) {
    defer http.conn.Close()
    resp := ""
    for i := 0; ; {
        data := make([]byte, 1)
        n, err := http.conn.Read(data)
        if err != nil {
            if err != io.EOF {
                fmt.Println("ERROR:", err.Error())
                defer http.conn.Close()
                http.Error = err
                return
            }
        }
        if data[0] == '\r' {
            continue
        } else if data[0] == '\n' {
            if i == 0 {
                break
            }
            i = 0
        } else {
            i++
        }
        resp += string(data[:n])
    }
    if http.Debug {
        fmt.Println("DEBUG:", resp)
    }

    for {
        data := make([]byte, buffer_size)
        n, err := http.conn.Read(data)
        if err != nil {
            return
        }
        f.WriteAt(data[:n], int64(http.offset))
        if http.Callback != nil {
            http.Callback(n)
        }
        http.offset += n
    }
    return
}

/*
func (http *HTTP) WriteToFile(fileName string, range_from int) {
    defer http.conn.Close()
    resp := ""
    for i := 0; ; {
        data := make([]byte, 1)
        n, err := http.conn.Read(data)
        if err != nil {
            if err != io.EOF {
                fmt.Println("ERROR:", err.Error())
                defer http.conn.Close()
                http.Error = err
                return
            }
        }
        if data[0] == '\r' {
            continue
        } else if data[0] == '\n' {
            if i == 0 {
                break
            }
            i = 0
        } else {
            i++
        }
        resp += string(data[:n])
    }
    if http.Debug {
        fmt.Println("DEBUG:", resp)
    }

    for {
        data := make([]byte, buffer_size)
        n, err := http.conn.Read(data)
        if err != nil {
            return
        }
        chunkName := fmt.Sprintf("%s.part.%d", fileName, range_from)
        f, err := os.Create(chunkName)
        if err != nil { panic(err) }
        defer f.Close()
        f.WriteAt(data[:n], int64(http.offset))
        if http.Callback != nil {
            http.Callback(n)
        }
        http.offset += n
    }
    return
}
*/

func (http *HTTP) Get(url string, range_from, range_to int) {
    http.offset = range_from
    http.AddHeader(fmt.Sprintf("GET %s HTTP/1.0", url))
    if range_to == 0 {
        http.AddHeader(fmt.Sprintf("Range: bytes=1-"))
    } else {
        http.AddHeader(fmt.Sprintf("Range: bytes=%d-%d", range_from, range_to))
    }
    http.AddHeader(fmt.Sprintf("User-Agent: %s", http.UserAgent))
    if http.Debug {
        fmt.Println("DEBUG:", http.header)
    }
    _, http.Error = http.conn.Write([]byte(http.header))
    if http.Error != nil {
        fmt.Println("ERROR: ", http.Error.Error())
    }
}

func (http *HTTP) IsAcceptRange() bool {
    ret := false

    if strings.Contains(http.headerResponse, "Accept-Ranges") {
        ret = true
    }

    return ret
}

func (http *HTTP) GetContentLength() int {
    ret := 0
    r, err := regexp.Compile(`Content-Length: (.*)`)
    if err != nil {
        http.Error = err
        fmt.Println("ERROR: ", err.Error())
        return ret
    }
    result := r.FindStringSubmatch(http.headerResponse)
    if len(result) != 0 {
        s := strings.TrimSuffix(result[1], "\r")
        ret, http.Error = strconv.Atoi(s)
    }
    return ret
}
