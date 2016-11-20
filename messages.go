package main

/*
 * List requests from player
 */

/* PlayerToListMessage is the base structure for requests to a baps3d list from a player handle.
   These inform the list of changes in the current playing file.
   These do not include messages routed directly to the player from a client. */
type PlayerToListMessage struct {
	/* Tag is the unique ID of the request that triggered this response. */
	Tag string
}

/*
 * List responses to player
 */

/*
 * List requests from client
 */

/*
 * List responses to client
 */
