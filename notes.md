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
and the cursor pagination logic for ratings is currently set up in a way that means `next_cursor` will never be returned. - `COMPLETED`

9. The media fetch fallback is wrong. The media handler is checking for the wrong error when media does not exist locally,
so the intended "fetch from TMDB and save it" path will not run as expected. - `COMPLETED`

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


### Recommended order to pick next
Balanced for importance + not-too-difficult, based on the current backend shape.

1. `#8` Ratings read endpoints are incomplete/incorrect.
High importance and likely one of the fastest wins. The API already exists, and this looks like a correctness fix in mapping/pagination rather than a new feature.

2. `#9` Media fetch fallback is wrong.
Also a strong, contained bug fix. The fetch-and-save path already exists, but the handler is checking the wrong error, so this should be straightforward and user-visible.

3. `#5` Login flow is not fully complete. Refresh tokens are missing.
Authentication is product-critical. This is more work than `#8` and `#9`, but still more bounded than the feed/search/matches items.

#### My notes for #3
create the new migration(just write the file, dont apply). I will give a mental model of what I assume this would look like without the replay stuff(family_id and replaced with). 1. User logins - a new authorization token is created(like it is now) and a new refresh token is also created, both
  with different expiries are returned to the client. 2. User accesses the app after an hour(auth token - 15m expiry), the client calls an endpoint like
  the rating one, sees that the auth token is invalid and calls the refresh endpoint. 3. The refresh endpoint checks the token(refresh - from the header)
  that it previously got from the client. The server verifies that this token has not been revoked and has not expired yet(60 day) - if that is the case, a new auth token is created and given back to the client to which it can use and make the same request again.

added notes on family_id
 You create a new family_id when a new login session starts.

  Concretely:

  - user logs in with email/password
  - you mint a refresh token
  - that refresh token starts a new family
  - every later rotated refresh token from that same session keeps the same family_id

  So:

  That is the point of the field: it models one long-lived refresh session across rotations.

  Typical lifecycle:

  - server creates R2
  - R2 keeps the same family_id

  3. Replay detected

  - old R1 is used again after rotation
  - revoke all non-revoked refresh tokens with that family_id

  So the rule is:
  If you only allow one session per user, you could avoid family_id and revoke all refresh tokens for that user on login. But if you want multiple devices/
  sessions, family_id is the clean boundary.

4. `#14` Define the rating translation layer (10->5) before sending results to clients.
Important for API consistency and likely not very large if applied cleanly at the response boundary.

5. `#13` Make watchlist behavior match the intended product buckets/states.
The CRUD surface already exists, so this is an iterative product-shaping task instead of a greenfield system.

6. `#7` Add the random movie/show rating endpoint for taste building.
Useful product feature with a fairly clear endpoint shape. Medium effort, but much more bounded than discovery feed work.

7. `#3` Wire matches into an endpoint or scheduled job.
The matching logic already exists, so exposing/triggering it is likely easier than redesigning the algorithm itself.

8. `#12` Finish the Impact feature backend.
Important for the UI, but this likely expands the reaction model and adds aggregation work, so it is less contained.

9. `#10` Expand search beyond TMDB movies/TV.
Good product value, but it cuts across TMDB search, user/match search, and possibly recent-search persistence, so scope can grow quickly.

10. `#6` Change the logging strategy to `slog`.
Reasonable cleanup, but not urgent and not directly user-facing.

11. `#2` Fix the `O(N^2)` matches solution.
Important long-term, but this is a performance/algorithm task that is easy to underestimate and not the best next issue if you want momentum.

12. `#11` Build the home/discovery feed backend surface.
High value, but this is one of the broadest items because it implies multiple endpoints, aggregations, and recommendation logic.

13. `#1` Implement multi-threaded comments in the data layer and add endpoints.
Large feature with schema, data-model, and API work. This is not a good "next issue" if the goal is importance plus manageable difficulty.

14. `#4` User profile page.
Already completed.
