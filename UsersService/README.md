# User service notes

## Enqueuing tasks - Workers

- This executes a request to the application.
- Information sent to the application is registered in two places when it comes to POST/PUT/DELETE. The first is the cache, where it will be all the information sent in the request. The second place is the queue, where it will be an identification key to the cached information.
- The workers start and check if there is some content in the queue.
- If there is content in the queue, the workers seek the data in the cache.
- After searching the data in the cache, the data is persisted in the database.
- This step is only for when the request is a GET. In this case, step 2 will only cache to fetch the data. If you can't find the data in the cache, step 6 runs by going to the database for the search. To return data, the searches are recorded in the cache.
