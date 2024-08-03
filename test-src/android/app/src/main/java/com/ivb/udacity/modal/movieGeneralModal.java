package com.ivb.udacity.modal;

import java.io.Serializable;

/**
 * Created by S.Shivasurya on 1/1/2016 - androidStudio.
 */
public class movieGeneralModal implements Serializable {
    String mTitle;
    String mThumbnail;
    String mVote;
    String mId;
    String mPeople;
    String mReleaseDate;
    String mOverview;
    String mReview;

    public movieGeneralModal(String mTitle, String mThumbnail, String mVote, String mId, String mPeople, String mReleaseDate, String mOverview) {
        this.mThumbnail = mThumbnail;
        this.mTitle = mTitle;
        this.mVote = mVote;
        this.mId = mId;
        this.mPeople = mPeople;
        this.mReleaseDate = mReleaseDate;
        this.mOverview = mOverview;
    }

    public String getmReview() {
        return this.mReview;
    }

    public void setmReview(String mReview) {
        this.mReview = mReview;
    }

    public String getmOverview() {
        return this.mOverview;
    }

    public String getmReleaseDate() {
        return this.mReleaseDate;
    }

    public String getTitle() {
        return this.mTitle;
    }

    public String getThumbnail() {
        String url = "http://image.tmdb.org/t/p/w185/" + this.mThumbnail;
        return url;
    }

    public String getmId() {
        return this.mId;
    }

    public String getmPeople() {
        return this.mPeople;
    }
    public String getmVote() {
        return this.mVote;
    }
}
