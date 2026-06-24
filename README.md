# SOUNDCLASH BACKEND

## Description
Soundclash is a web application powered by Golang, where music producers can participate in multiplayer timed battles to produce the best track within a time limit, and then rate each others work and vote for the winner at the end. This backend repo is responsible for managing users, the battle schedular, leaderboards, matchmaking, and audio uploads & downloads. 
<img width="1904" height="897" alt="image" src="https://github.com/user-attachments/assets/6e20c336-4917-4ec8-8f18-bd28967a2bd5" />

## Motivation
As a electronic music enthusiast i noticed a similar platform gaining traction online, but found this platform to be clunky, unresponsive, poorly optimised, and heavily focused on a neiche genre of music that is not to my taste. This project is inspired by a desire to create a more robust and hollistic platform for the same purpose, that would scale to handle thousands of users and concurrent battles. There are still plenty of features that would make this platform better, and I hope that in time i will be able to improve and iterate on this project so it is suitable for public release and even sponsored events (I'm looking at you, Red Bull). This is a platform for creatives to share their passion, improve their creative works, and compete for leaderboard titles to be the best producer on the internet!

## Quick Start
To access the application, navigate to: https://soundclash-three.vercel.app/ - create an account, and you're live! 
You will notice a significant front end design is in place that i must credit mostly to Claude, and my ability to prompt a responsive design together. That being said, the UI and UX is simple and easy to navigate once you have created an account and logged in. 

## Usage
Due to a lack of users, you will likely struggle to actually partake in a battle unless you also find a friend (or rival!) to sign up and play with you. The best way to play is to create a custom Lobby, select your game options, and have them join your battle.

<img width="536" height="691" alt="image" src="https://github.com/user-attachments/assets/5a143505-9b55-4e70-bc86-aa2c13c0c7f7" />

The system runs on a schedular with webhooks that keep all users in sync, battles managed properly, and accounts for a lot of challenging edge cases that actually took up the majority of the development time overcoming. The systems current state is very robust and accounts for lots of edge case scenarios such as temporary disconnects, forfeits, oversize file uploads, browser refresh issues, and keeping all participants in sync to a tight battle schedule, so they can all experience the battle concurrently, with no latency.

## Contributing
While this repository is public, please make contact to discuss contributing to the project before cloning or making any pull requests. This is a personal one man band project to date, and I have a vision for its design. There are lots of areas open for collaboration, so if you're interested in joing the team, please reach out to devjoshbrown@gmail.com.

