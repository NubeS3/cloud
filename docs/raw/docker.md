# Docker

## Sơ lược 

### Cơ bản

-   Docker là một công cụ giúp cho việc tạo ra và triển khai các **container** để phát triển, chạy ứng dụng được dễ dàng. Các **container** là môi trường, mà ở đó lập trình viên đưa vào các thành phần cần thiết để ứng dụng của họ chạy được, bằng cách đóng gói ứng dụng cùng với container như vậy, nó đảm bảo ứng dụng **chạy được và giống nhau** ở các máy khác nhau (Linux, Windows, Desktop, Server ...).
    
-   Hình ảnh vùng chứa Docker (**Docker Image**) là một gói phần mềm nhẹ, độc lập, có thể thực thi bao gồm mọi thứ cần thiết để chạy một ứng dụng: mã, thời gian chạy, công cụ hệ thống, thư viện hệ thống và cài đặt. Docker Image trở thành container khi được khởi chạy trên **Docker Engine**.
    
-   Docker cho phép sử dụng nhân của hệ điều hành để chạy ứng dụng, bằng cách sử dụng các thành phần cần thiết còn thiếu được cung cấp trong container được đóng gói. => Đem lại hiệu năng tốt hơn Virtual Machine (máy ảo), cũng như tận dụng tốt tài nguyên.
    
-   Tương thích tốt với các nền tảng, tốt nhất trên Linux (phù hợp triển khai ở server).


### Vì sao là Docker?

-	Miễn phí.

-	Giúp lập trình viên tập trung vào giải quyết các vấn đề chính, giảm thời gian thiết lập các môi trường.

-	Đem lại môi trường đồng nhất giữa hệ thống phát triển và hệ thống triển khai => dễ dàng phát hiện các lỗi, vấn đề.

-	Các vùng chứa nhẹ, giảm tải, giảm tài nguyên,  tăng hiệu suất máy chủ.
-	Tương thích với các hệ điều hành phổ biến. Tương thích tốt với Linux => Dễ dàng triển khai trên Server.

> Tham khảo: [What is a Container?](https://www.docker.com/resources/what-container)



## Xây dựng và vận hành Docker image

Có 2 cách tạo để tạo ra Docker image: *Thủ công* và sử dụng **Dockerfile**. 

>Tham khảo: [Bắt đầu với Docker (1 số thao tác cơ bản và bắt đầu tạo image)](https://docs.docker.com/get-started/https://docs.docker.com/get-started/)

	Yêu cầu: đã hoàn thành các bước cài đặt và thiết lập cơ bản Docker.

Đối với việc tạo ra Docker image bằng việc thiết lập Dockerfile đòi ta phải xác định cũng như lưu ý một vài điểm:
-	Xác định được môi trường ứng dụng sẽ chạy => lựa chọn được base image.
- Tiến hành setup *working directory*.
- Copy các file cần thiết cho ứng dụng.
- Xác định command cần chạy cho image filesystem (*dùng với lệnh `RUN`*).
- Xác định cổng hoạt động của container. 
- Xác định command đặc biệt để chạy khi container khởi chạy (*lệnh `CMD`*).
- Xác định cách thức map các cổng giữa container và host (máy thật).

Lệnh khởi chạy 1 container sẽ có dạng như sau:
```Docker
sudo docker run -v <forder_in_computer>:<forder_in_container> -p <port_in_computer>:<port_in_container> -it <image_name> /bin/bash
```


## Switch to another file 

All your files and folders are presented as a tree in the file explorer. You can switch from one to another by clicking a file in the tree.

## Rename a file

You can rename the current file by clicking the file name in the navigation bar or by clicking the **Rename** button in the file explorer.

## Delete a file

You can delete the current file by clicking the **Remove** button in the file explorer. The file will be moved into the **Trash** folder and automatically deleted after 7 days of inactivity.

## Export a file

You can export the current file by clicking **Export to disk** in the menu. You can choose to export the file as plain Markdown, as HTML using a Handlebars template or as a PDF.






> Before starting to publish, you must link an account in the **Publish** sub-menu.

## Publish a File

You can publish your file by opening the **Publish** sub-menu and by clicking **Publish to**. For some locations, you can choose between the following formats:

- Markdown: publish the Markdown text on a website that can interpret it (**GitHub** for instance),
- HTML: publish the file converted to HTML via a Handlebars template (on a blog for example).

## Update a publication

After publishing, StackEdit keeps your file linked to that publication which makes it easy for you to re-publish it. Once you have modified your file and you want to update your publication, click on the **Publish now** button in the navigation bar.

> **Note:** The **Publish now** button is disabled if your file has not been published yet.

## Manage file publication

Since one file can be published to multiple locations, you can list and manage publish locations by clicking **File publication** in the **Publish** sub-menu. This allows you to list and remove publication locations that are linked to your file.


# Markdown extensions

StackEdit extends the standard Markdown syntax by adding extra **Markdown extensions**, providing you with some nice features.

> **ProTip:** You can disable any **Markdown extension** in the **File properties** dialog.


## SmartyPants

SmartyPants converts ASCII punctuation characters into "smart" typographic punctuation HTML entities. For example:

|                  | ASCII                           | HTML                          |
| ---------------- | ------------------------------- | ----------------------------- |
| Single backticks | `'Isn't this fun?'`             | 'Isn't this fun?'             |
| Quotes           | `"Isn't this fun?"`             | "Isn't this fun?"             |
| Dashes           | `-- is en-dash, --- is em-dash` | -- is en-dash, --- is em-dash |


## KaTeX

You can render LaTeX mathematical expressions using [KaTeX](https://khan.github.io/KaTeX/):

The *Gamma function* satisfying $\Gamma(n) = (n-1)!\quad\forall n\in\mathbb N$ is via the Euler integral

$$
\Gamma(z) = \int_0^\infty t^{z-1}e^{-t}dt\,.
$$

> You can find more information about **LaTeX** mathematical expressions [here](http://meta.math.stackexchange.com/questions/5020/mathjax-basic-tutorial-and-quick-reference).


## UML diagrams

You can render UML diagrams using [Mermaid](https://mermaidjs.github.io/). For example, this will produce a sequence diagram:

```mermaid
sequenceDiagram
Alice ->> Bob: Hello Bob, how are you?
Bob-->>John: How about you John?
Bob--x Alice: I am good thanks!
Bob-x John: I am good thanks!
Note right of John: Bob thinks a long<br/>long time, so long<br/>that the text does<br/>not fit on a row.

Bob-->Alice: Checking with John...
Alice->John: Yes... John, how are you?
```

And this will produce a flow chart:

```mermaid
graph LR
A[Square Rect] -- Link text --> B((Circle))
A --> C(Round Rect)
B --> D{Rhombus}
C --> D
```