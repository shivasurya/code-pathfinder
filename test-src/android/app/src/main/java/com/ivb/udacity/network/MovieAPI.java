package com.ivb.udacity.network;

import com.ivb.udacity.modal.movieGeneral;
import com.ivb.udacity.modal.review.movieReview;
import com.ivb.udacity.modal.trailer.movieYoutubeModal;

import retrofit.Callback;
import retrofit.http.GET;
import retrofit.http.Path;
import retrofit.http.Query;

/**
 * Created by S.Shivasurya on 1/1/2016 - androidStudio.
 */
public interface MovieAPI {

    //this method is to fetch the ALL movies with specific sort
    @GET("/3/discover/movie")
    void fetchMovies(
            @Query("sort_by") String mSort,
            @Query("api_key") String mApiKey,
            @Query("language") String lang,
            Callback<movieGeneral> cb
    );

    @GET("/3/movie/{id}/reviews")
    void fetchReview(
            @Query("api_key") String mApiKey,
            @Path("id") String id,
            Callback<movieReview> cb
    );

    @GET("/3/movie/{id}/videos")
    void fetchVideos(
            @Query("api_key") String mApiKey,
            @Path("id") String id,
            Callback<movieYoutubeModal> cb
    );

}
