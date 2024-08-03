package com.ivb.udacity.modal;

import java.io.Serializable;

/**
 * Created by S.Shivasurya on 1/1/2016 - androidStudio.
 */
public class movieGeneral implements Serializable {
    private Results[] results;

    private String page;

    private String total_pages;

    private String total_results;

    public Results[] getResults() {
        return results;
    }

    public void setResults(Results[] results) {
        this.results = results;
    }

    public String getPage() {
        return page;
    }

    public void setPage(String page) {
        this.page = page;
    }

    public String getTotal_pages() {
        return total_pages;
    }

    public void setTotal_pages(String total_pages) {
        this.total_pages = total_pages;
    }

    public String getTotal_results() {
        return total_results;
    }

    public void setTotal_results(String total_results) {
        this.total_results = total_results;
    }

    @Override
    public String toString() {
        return "ClassPojo [results = " + results + ", page = " + page + ", total_pages = " + total_pages + ", total_results = " + total_results + "]";
    }
}
