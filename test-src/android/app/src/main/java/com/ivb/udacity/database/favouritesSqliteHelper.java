package com.ivb.udacity.database;

import android.content.ContentValues;
import android.content.Context;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.database.sqlite.SQLiteOpenHelper;

import com.ivb.udacity.modal.movieGeneralModal;

import java.util.ArrayList;
import java.util.List;

/**
 * Created by S.Shivasurya on 1/11/2016 - androidStudio.
 */
public class favouritesSqliteHelper extends SQLiteOpenHelper {
    public static final String KEY_ROWID = "id";
    public static final String KEY_THUMBNAIL = "mThumbnail";
    public static final String KEY_MVOTE = "mVote";
    public static final String KEY_TITLE = "mTitle";
    public static final String KEY_PEOPLE = "mPeople";
    public static final String KEY_RELEASEDATE = "mReleaseDate";
    public static final String KEY_OVERVIEW = "mOverview";
    public static final String KEY_REVIEW = "mReview";
    public static final String SQLITE_TABLE = "movies";
    private static final String LOG_TAG = "moviesDB";
    private static final String DATABASE_CREATE =
            "CREATE TABLE if not exists " + SQLITE_TABLE + " (" +
                    KEY_ROWID + " integer PRIMARY KEY," +
                    KEY_THUMBNAIL + "," +
                    KEY_TITLE + "," +
                    KEY_PEOPLE + "," +
                    KEY_MVOTE + "," +
                    KEY_OVERVIEW + "," +
                    KEY_REVIEW + "," +
                    KEY_RELEASEDATE + "" +
                    " );";

    public favouritesSqliteHelper(Context context) {
        super(context, LOG_TAG, null, 1);
    }


    @Override
    public void onCreate(SQLiteDatabase db) {
        db.execSQL(DATABASE_CREATE);
    }

    @Override
    public void onUpgrade(SQLiteDatabase db, int oldVersion, int newVersion) {
        db.execSQL("DROP TABLE IF EXISTS " + SQLITE_TABLE);
        onCreate(db);
    }

    public boolean insertMovie(movieGeneralModal movieGeneralModals) {
        SQLiteDatabase db = this.getWritableDatabase();
        ContentValues values = new ContentValues();
        values.put(KEY_ROWID, Integer.parseInt(movieGeneralModals.getmId()));
        values.put(KEY_THUMBNAIL, movieGeneralModals.getThumbnail());
        values.put(KEY_TITLE, movieGeneralModals.getTitle());
        values.put(KEY_PEOPLE, movieGeneralModals.getmPeople());
        values.put(KEY_MVOTE, movieGeneralModals.getmVote());
        values.put(KEY_OVERVIEW, movieGeneralModals.getmOverview());
        values.put(KEY_RELEASEDATE, movieGeneralModals.getmReleaseDate());
        values.put(KEY_REVIEW, movieGeneralModals.getmReview());

        boolean createSuccessful = db.insert(SQLITE_TABLE, null, values) > 0;
        db.close();
        return createSuccessful;
    }

    public List<movieGeneralModal> getAllMovies() {
        List<movieGeneralModal> movieList = new ArrayList<>();
        String selectQuery = "SELECT  * FROM " + SQLITE_TABLE;

        SQLiteDatabase db = this.getReadableDatabase();
        Cursor cursor = db.rawQuery(selectQuery, null);

        if (cursor.moveToFirst()) {
            do {
                movieGeneralModal movie = new movieGeneralModal(cursor.getString(2), cursor.getString(1), cursor.getString(4), cursor.getString(0), cursor.getString(3), cursor.getString(7), cursor.getString(5));
                movieList.add(movie);
            } while (cursor.moveToNext());
        }
        cursor.close();

        return movieList;
    }

}
