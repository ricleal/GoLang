# create book
http -v POST localhost:8080/books <<< '{
    "id": 1,
    "title": "The Great Gatsby",
    "author": "F. Scott Fitzgerald"
}'
# create another book
http -v POST localhost:8080/books <<< '{
    "id": 2,
    "title": "Era of Ignition",
    "author": "Amber Tamblyn"
}'
# get all books
http -v GET localhost:8080/books
# get a book by id
http -v GET localhost:8080/books/1
# update a book
http -v PUT localhost:8080/books/1 <<< '{
    "title": "The Great Gatsby",
    "author": "Scott Fitzgerald"
}'
# delete a book
http -v DELETE localhost:8080/books/1
# get all books
http -v GET localhost:8080/books

