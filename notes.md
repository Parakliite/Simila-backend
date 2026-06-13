### This is my personal reference for what is missing in the project so far
Author: deltron-fr


1. I haven't implemented multi-threaded comments on the data layer. It is only single-threaded for now and 
 it endpoints for it haven't been created.

2. The current solution for matches is O(N^2). At scale that is horrendous. I need to remember to fix that.

3. Currently no endpoint calls the matches functionality. It currently lives it on its own. Though the plan for it is
  to use a cronjob or some kind of scheduler

4. The users profile page has not yet been implemented - `COMPLETED`

5. Login flow is not fully complete. Refresh tokens are completely missing

6. This is not really important right now but change the logging strategy to slog.

7. There should be a dedicated endpoint that calls random movies/shows(returns about 5) but shows only 1 to the user for them to 
rate - something like tiktok videos. This will be used to build the users taste over time


### Additional missing / incorrect items found while comparing against the UI

8. The ratings read endpoints are incomplete/incorrect. The list/get rating responses are not populating `rating_value`,
and the cursor pagination logic for ratings is currently set up in a way that means `next_cursor` will never be returned.

9. The media fetch fallback is wrong. The media handler is checking for the wrong error when media does not exist locally,
so the intended "fetch from TMDB and save it" path will not run as expected.

10. Search is only covering TMDB movies and TV shows right now. The UI also searches people/matches and has search-related
discovery affordances, so there is no backend support yet for searching users/matches or persisting recent searches if I keep that behavior.

11. There is no backend surface for the home/discovery feed that the UI expects. Things like "top rated by your matches",
"because you loved ...", "pick of the night", "hot takes", and the general activity feed still need proper backend endpoints or aggregators.

12. The Impact feature is only partially backed. Raw reactions exist, but there is no aggregated Impact endpoint for the summary
counts/highlights feed, and the current reaction model cannot store the watched follow-up sentiment or optional quote shown in the UI.

13. The watchlist endpoints still need to better match the product behavior I actually want. The current model can support this,
but I still need to shape the filtering/state behavior around things like the watchlist buckets and how watched/not-watched items
should be returned to the client.

14. I need to define a proper translation layer(10->5) for ratings before sending results out to the clients.
