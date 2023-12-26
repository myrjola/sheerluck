------------------------------ MODULE completion -----------------------------
(*
Spec for checking robustness in AI completions.

Requirements:
* Given a completion history, only one completion can complete successfully for a single user and target at a time.
  This spec simplifies the user and target away, i.e., we assume that any identifiers used in
  HTTP requests target the same user and target. For example, user comes from session cookie and target is passed
  in the request.
* The database and UI should stay in sync

Happy path:
1. User starts completion given history, the prompt is stored to the history
2. User requests to stream the completion by a GET to SSE event source
3. Server requests streaming completion from AI provider and streams it to the user
3. The completion streams to the user
4. Once the completion is done, the server requests user to close the stream and persists the completion

Open Questions:
* When is the completion post-processed, e.g., detecting clues, highlighting character names?
  * Probably more complex to implement as part of streaming, but better UX for the user
  * Other option is to do post-processing after the completion and request UI to refresh the completion once it's
    enriched
* How to handle errors keeping UI in sync
* Can we support progressive enhancement?
  * GET is the same body for both hx-boost and plain HTML. No change necessary.
  * plain POST performs completion and responds with same response as a get
  * hx-post initiates the happy path

Error scenarios:
* AI provider is down
* server crashes at any point
* user closes the browser?
* user reloads the page during streaming
* user requests a new completion while another is in progress or completed

Approach:

* The completions table maintains state with the following columns:
  * id: unique identifier for the completion
  * user_id: the user who requested the completion
  * target_id: the target for the completion
  * order: the position in the completion history
  * status: the status of the completion (see below)
  * prompt: the prompt of the completion
  * completion: the completion of the completion
* Server responds with HTMX to keep UI state in sync
* UI reacts to errors by showing error message and requesting user to retry

Completion statuses:
* created: the completion is created and ready to be streamed
* streaming: the completion is being streamed to the user
* done: the completion is done and ready to be persisted

*)
EXTENDS TLC, FiniteSets

CONSTANTS
  Completion, \* the set of all possible completions
  UNDEFINED, \* not created
  NULL \* empty value

ASSUME Completion # {}
ASSUME NULL \notin Completion

VARIABLES
  completionStatus, \* the status of the completion
  completionParent \* the parent completion

vars == <<completionStatus, completionParent>>

-----------------------------------------------------------------------------

CompletionStatusType == {"not-created", "created", "streaming", "done"}

RangeStruct(struct) == {struct[key]: key \in DOMAIN struct}

TypeOK ==
    /\ \A c \in Completion: completionStatus[c] \in CompletionStatusType
    /\ \A c \in Completion: completionParent[c] \in Completion \cup {NULL, UNDEFINED}

Init ==
  /\ completionStatus = [c \in Completion |-> "not-created"]
  /\ completionParent = [c \in Completion |-> UNDEFINED]

CreateCompletion ==
  /\ \E completion \in Completion:
    /\ completionStatus[completion] = "not-created"
    /\ completionStatus' = [completionStatus EXCEPT ![completion] = "created"]
    /\ UNCHANGED <<completionParent>>

Next ==
  \/ CreateCompletion

Fairness ==
  /\ WF_vars(UNCHANGED vars)

Spec == Init /\ [][Next]_vars /\ Fairness
-----------------------------------------------------------------------------

ContiguousHistory ==
    /\ \/ \A c \in Completion: completionStatus[c] = "not-created"
       \/ Cardinality({ c \in Completion : completionParent[c] = NULL }) = 1
    /\ \A c1, c2 \in Completion:
      \/  completionParent[c1] \in {NULL, UNDEFINED}
      \/  completionParent[c1] # completionParent[c2]

Safety ==
  [](ContiguousHistory)

Liveness == <>Init
=============================================================================