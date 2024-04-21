<div align="center">
  <img src="./assets/pathfinder-logo.png" alt="Code Pathfinder" width="75" height="75"/>
</p>

# Code Pathfinder 
Code Pathfinder attempts to be query language for structural search on source code. It's built for identifying vulnerabilities in source code.

[![Build and Release](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml/badge.svg)](https://github.com/shivasurya/code-pathfinder/actions/workflows/build.yml)
</div>

## Getting Started
Read the [documentation](./README.md), or run `pathfinder --help`.

## Features

- [x] Basic queries
- [x] Source Sink Analysis
- [ ] Taint Analysis
- [ ] Data Flow Analysis with Control Flow Graph

## Usage

Currently tested on Mac & Linux. Use `docker-compose` for Windows.

```bash
$ cd sourcecode-parser

$ go build -o pathfinder

$ ./pathfinder /PATH/TO/SOURCE

2024/04/19 12:46:08 Graph built successfully
Path-Finder Query Console: 
>FIND method WHERE name = 'onCreate'
FIND method WHERE name = 'onCreate'
------Results------
@Override
public void onCreate(SQLiteDatabase db) {
    db.execSQL(DATABASE_CREATE);
}
-------
@Override
protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);
    setContentView(R.layout.activity_movie_detail);
    Intent intent = getIntent();

    getSupportActionBar().setDisplayHomeAsUpEnabled(true);
    getSupportActionBar().setDisplayShowHomeEnabled(true);

    movieGeneralModal moviegeneralModal = (movieGeneralModal) intent.getSerializableExtra("DATA_MOVIE");

    if (savedInstanceState == null) {

        movieDetailFragment fragment = new movieDetailFragment();
        fragment.setMovieData(moviegeneralModal);
        getSupportFragmentManager().beginTransaction()
                .add(R.id.movie_detail_container, fragment)
                .commit();
    }
}
------Results------
```

## Acknowledgements
Code Pathfinder uses tree-sitter for all language parsers.

