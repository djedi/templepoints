# Temple Points Tracker

Our stake is having a contest to see which ward can get to 1360 points the fastest. You get points by doing ordinances in the temple mainly.

There are 7 wards:

Fountain Green 1st Ward
Fountain Green 2nd Ward
Fountain Green 3rd Ward
Moroni 1st Ward
Moroni 2nd Ward
Moroni 3rd Ward
Sanpitch Ward

## Views

I want to create a website that will have the following views:

### Point Tracker

Show a leaderboard of each of the wards and their current number of points. It should show total verified points and total points + pending points. This is a public view. Anyone can see it and sort by ward, points, or points + pending. Default will be sorted by verified points descending.

There should be a link for each ward's point audit log

### Ward points log

For each ward, show a log of all their points. Each log entry should include date/time submitted, name, note, and number of points, and wheter is has been approved.

### Point Input form

Allow anyone to enter points. They would enter their name, ward, and number of points, and a note.

If a person enters points, save their ward and name in a cookie so it pre-populates the points form again in the future.

### Point approval form

Each ward will have one or more people who can approve or deny points that have been submitted. So they should see a list of points that have been submitted in their ward that they can verify and approve.

## Users

There will be 3 main types of users on this app:

1. General users: view points and submit points - no login required
2. Ward Point approvers: Can be general users, plus they have access to the points approval view to approve points for their ward
3. Stake Admin: Can view all pending points, can add ward point approvers.

Point approvers and stake admins should will need to be able to log in. Just use email and password for authentication.

## Tech stack

I want this app to be fast and small, so I would prefer to use Go because of its speed and low memory usage. Static pages can also be served with Go. If a templating language is helpful for the html, use the industry best practices for Go apps.

Use sqlite for the database

## Deployment

When deployed, it should run in a docker container and a docker compose file will be used with Caddy as a reverse proxy that will handle the SSL cert management automatically.

I prefer "one-click" deployment, so I would like a script named `deploy` so I can just run `./deploy`.

## Notes & Preferences

- Most people will view this on their phone, so use a responsive design and keep it mobile friendly
- This is for members of the Church of Jesus Christ of Latter-day Saints, but not affilicate with the church. Put a disclaimer that says such on the footer of each page.
- The look and feel of the views should be similar to chruchofjesuschrist.org
- I want real-time updates, so use websockets to update the views in real-time
