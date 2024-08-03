package com.ivb.udacity.adapter;

import android.content.Context;
import android.content.Intent;
import android.support.v4.app.FragmentManager;
import android.support.v7.widget.RecyclerView;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;

import com.ivb.udacity.R;
import com.ivb.udacity.modal.movieGeneralModal;
import com.ivb.udacity.movieDetailActivity;
import com.ivb.udacity.movieDetailFragment;
import com.squareup.picasso.Picasso;

import java.util.List;

/**
 * Created by S.Shivasurya on 1/1/2016 - androidStudio.
 */
public class movieGeneralAdapter extends RecyclerView.Adapter<movieGeneralHolder> {
    private List<movieGeneralModal> mMovieGeneralModal;
    private Context context;
    private boolean mTwoPane;
    private FragmentManager fm;

    public movieGeneralAdapter(Context context, List<movieGeneralModal> itemList, boolean mTwoPane, FragmentManager fm) {
        this.mMovieGeneralModal = itemList;
        this.context = context;
        this.mTwoPane = mTwoPane;
        this.fm = fm;
    }

    @Override
    public movieGeneralHolder onCreateViewHolder(ViewGroup parent, int viewType) {
        View layoutView = LayoutInflater.from(parent.getContext()).inflate(R.layout.movie_cards, null);
        movieGeneralHolder rcv = new movieGeneralHolder(layoutView);
        return rcv;
    }

    @Override
    public void onBindViewHolder(movieGeneralHolder holder, final int position) {
        holder.movieName.setText(mMovieGeneralModal.get(position).getTitle());
        holder.movieAvg.setText(mMovieGeneralModal.get(position).getmVote());
        //picasso loading here
        Picasso.with(context)
                .load(mMovieGeneralModal.get(position).getThumbnail())
                .into(holder.moviePhoto);
        if (position == 0 && mTwoPane) {
            movieDetailFragment fragment = new movieDetailFragment();
            fragment.setMovieData(mMovieGeneralModal.get(0));
            fragment.setArgument(fm);
            fm
                    .beginTransaction()
                    .replace(R.id.movie_detail_container, fragment)
                    .commit();
        }
        holder.mView.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                if (mTwoPane) {
                    movieDetailFragment fragment = new movieDetailFragment();
                    fragment.setMovieData(mMovieGeneralModal.get(position));
                    fragment.setArgument(fm);
                    fm
                            .beginTransaction()
                            .replace(R.id.movie_detail_container, fragment)
                            .commit();
                } else {
                    Context context = v.getContext();
                    Intent intent = new Intent(context, movieDetailActivity.class);
                    intent.putExtra("DATA_MOVIE", mMovieGeneralModal.get(position));
                    context.startActivity(intent);
                }
            }
        });
    }

    @Override
    public int getItemCount() {
        return this.mMovieGeneralModal.size();
    }
}


