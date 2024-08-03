package com.ivb.udacity.modal.review;

/**
 * Created by S.Shivasurya on 1/8/2016 - androidStudio.
 */
public class Results {
    private String content;

    private String id;

    private String author;

    private String url;

    public String getContent() {
        return content;
    }

    public void setContent(String content) {
        this.content = content;
    }

    public String getId() {
        return id;
    }

    public void setId(String id) {
        this.id = id;
    }

    public String getAuthor() {
        return author;
    }

    public void setAuthor(String author) {
        this.author = author;
    }

    public String getUrl() {
        return url;
    }

    public void setUrl(String url) {
        this.url = url;
    }

    @Override
    public String toString() {
        return "ClassPojo [content = " + content + ", id = " + id + ", author = " + author + ", url = " + url + "]";
    }
}
