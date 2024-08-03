package com.ivb.udacity.modal.trailer;

/**
 * Created by S.Shivasurya on 1/10/2016 - androidStudio.
 */
public class movieYoutubeModal {

    private String id;

    private Results[] results;

    public String getId() {
        return id;
    }

    public void setId(String id) {
        this.id = id;
    }

    public Results[] getResults() {
        return results;
    }

    public void setResults(Results[] results) {
        this.results = results;
    }

    @Override
    public String toString() {
        return "ClassPojo [id = " + id + ", results = " + results + "]";
    }
}
