package com.ivb.udacity.modal.trailer;

/**
 * Created by S.Shivasurya on 1/10/2016 - androidStudio.
 */
public class Results {
    private String site;

    private String id;

    private String iso_639_1;

    private String name;

    private String type;

    private String key;

    private String size;

    public String getSite() {
        return site;
    }

    public void setSite(String site) {
        this.site = site;
    }

    public String getId() {
        return id;
    }

    public void setId(String id) {
        this.id = id;
    }

    public String getIso_639_1() {
        return iso_639_1;
    }

    public void setIso_639_1(String iso_639_1) {
        this.iso_639_1 = iso_639_1;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }

    public String getKey() {
        return key;
    }

    public void setKey(String key) {
        this.key = key;
    }

    public String getSize() {
        return size;
    }

    public void setSize(String size) {
        this.size = size;
    }

    @Override
    public String toString() {
        return "ClassPojo [site = " + site + ", id = " + id + ", iso_639_1 = " + iso_639_1 + ", name = " + name + ", type = " + type + ", key = " + key + ", size = " + size + "]";
    }
}
