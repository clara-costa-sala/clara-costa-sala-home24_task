1. Clone the repository
2. Execute the executable file 
```./webInfo```
3. Go to the browser and write `http://127.0.0.1:8080`
4. Write an URL with https:// and wait until the reponse of the information of the website URL appears

Information that should appear: 
- HTML Version
- Page Title
- Headings count by level
- Amount of internal and external links
- Amount of inaccessible links
- If a page contains a login form

Note: For internal links I considered that if we are checking https://example.com, if there is a link with https://example.com it is internal. 
