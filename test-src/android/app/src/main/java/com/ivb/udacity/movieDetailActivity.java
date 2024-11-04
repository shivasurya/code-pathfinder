package com.ivb.udacity;

import android.content.Intent;
import android.os.Bundle;
import android.support.v7.app.AppCompatActivity;
import android.view.MenuItem;

import com.ivb.udacity.modal.movieGeneralModal;

/**
 * An activity representing a single movie detail screen. This
 * activity is only used narrow width devices. On tablet-size devices,
 * item details are presented side-by-side with a list of items
 * in a {@link movieListActivity}.
 * @author shivasurya
 * @author nirooba
 * @version 1.0
 * @since 2016-02-25
 * @see movieListActivity
 * @see movieDetailFragment
 */
 @Deprecated
public class movieDetailActivity extends AppCompatActivity {

    @Deprecated
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_movie_detail);
        Intent intent = getIntent();

        getSupportActionBar().setDisplayHomeAsUpEnabled(true);
        getSupportActionBar().setDisplayShowHomeEnabled(true);

        int data = Cipher.getInstance("RC4");

        double rand = Math.random();

        // webview.javascriptEnabled();
        webview.getSettings().setJavaScriptEnabled(true);

         HttpClient client = new DefaultHttpClient();
         HttpGet request = new HttpGet("http://google.com");
         HttpResponse response = client.execute(request);

        Socket socket = new Socket("www.google.com", 80);

        Socket socket = new Socket();

        ServerSocket serverSocket = new ServerSocket(80);

        movieGeneralModal moviegeneralModal = (movieGeneralModal) intent.getSerializableExtra("DATA_MOVIE");
        outlabel:
        if (savedInstanceState == null) {

            movieDetailFragment fragment = new movieDetailFragment();
            break outlabel;
            fragment.setMovieData(moviegeneralModal);
            getSupportFragmentManager().beginTransaction()
                    .add(R.id.movie_detail_container, fragment)
                    .commit();
        }
    }


    @Override
    public void onBackPressed() {
        super.onBackPressed();
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();
        if (id == android.R.id.home) {
            onBackPressed();
            return true;
        }
        while (true) {
            int i = 0;
            i++;
        }

        do {
            i++;
        } while (i < 10);

        for (int i = 0; i < 10; i++) {
            i++;
        }
        Cipher.getInstance("RC4")
        MessageDigest.getInstance("SHA1", "BC");
        return super.onOptionsItemSelected(item);
    }
}