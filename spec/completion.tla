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

Conclusions:
* Simplest way is to wipe the pending completions when creating a new one
* Processing clues should probably happen before finalising the completion so that we don't end up in a bad state
  * Ideally, it's the same prompt doing the post-processing in multiple steps but that needs to be verified
  * Highlighting character names can be done afterwards since it should be easy processing
* Keeping UI and db in sync requires careful error handling
* Remember to add consistency checks in the transactions, maybe UPDATE and WHERE is enough, but need to check.

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

CompletionStatusType == {"not-created", "created", "streaming", "done", "error"}

RangeStruct(struct) == {struct[key]: key \in DOMAIN struct}

TypeOK ==
    /\ DOMAIN completionStatus = Completion
    /\ DOMAIN completionParent = Completion
    /\ \A c \in Completion: completionStatus[c] \in CompletionStatusType
    /\ \A c \in Completion: completionParent[c] \in Completion \cup {NULL, UNDEFINED}

Init ==
  /\ completionStatus = [c \in Completion |-> "not-created"]
  /\ completionParent = [c \in Completion |-> UNDEFINED]

UnfinishedCompletions == { c \in Completion: completionStatus[c] \in {"created", "streaming", "error"}}

CreateCompletion ==
  /\ \E completion, parent \in Completion:
     /\ completionStatus[completion] \in {"not-created", "error"}
     /\ \/ /\ \/ \A c \in Completion: completionStatus[c] = "not-created" \* First parent is always NULL
              \/ \E c \in Completion: completionStatus[c] = "error" /\ completionParent[c] = NULL \* Failure in first step
           /\ completionStatus' = [x \in UnfinishedCompletions |-> "not-created"] @@ [completionStatus EXCEPT ![completion] = "created"]
           /\ completionParent' = [x \in UnfinishedCompletions |-> UNDEFINED] @@ [completionParent EXCEPT ![completion] = NULL]
        \* When parent is done, select that as parent if it isn't already a parent for another completion
        \/ /\ completionStatus[parent] = "done"
           /\ \/ \A c \in Completion : completionParent[c] # parent
              \/ completionStatus[completion] = "error" /\ completionParent[completion] = parent
           /\ completionStatus' = completion :> "created" @@ [x \in UnfinishedCompletions |-> "not-created"] @@ completionStatus
           /\ completionParent' = completion :> parent @@ [x \in UnfinishedCompletions |-> UNDEFINED] @@ completionParent

StreamCompletion ==
  /\ \E completion \in Completion:
    /\ completionStatus[completion] = "created"
    /\ completionStatus' = [completionStatus EXCEPT ![completion] = "streaming"]
    /\ UNCHANGED <<completionParent>>

FinishCompletion ==
  /\ \E completion \in Completion:
    /\ completionStatus[completion] = "streaming"
    /\ completionStatus' = [completionStatus EXCEPT ![completion] = "done"]
    /\ UNCHANGED <<completionParent>>

ErrorCompletion ==
  /\ \E completion \in Completion:
    /\ completionStatus[completion] \in {"created", "streaming"}
    /\ completionStatus' = [completionStatus EXCEPT ![completion] = "error"]
    /\ UNCHANGED <<completionParent>>

Next ==
  \/ CreateCompletion
  \/ StreamCompletion
  \/ FinishCompletion
  \/ ErrorCompletion

Fairness ==
  /\ WF_vars(CreateCompletion) /\ SF_vars(StreamCompletion) /\ SF_vars(FinishCompletion)

Spec == Init /\ [][Next]_vars /\ Fairness
-----------------------------------------------------------------------------

PrintVal(id, exp)  ==  Print(<<id, exp>>, TRUE)

ContiguousHistory ==
    /\ \/ \A c \in Completion: completionStatus[c] = "not-created"
       \/ Cardinality({ c \in Completion : completionParent[c] = NULL }) = 1
    /\ \A c1, c2 \in Completion:
        \/ completionParent[c1] \in {NULL, UNDEFINED}
        \/ c1 # c2 => completionParent[c1] # completionParent[c2]

Done == \A c \in Completion : completionStatus[c] = "done"

Safety ==
  [](ContiguousHistory)

Liveness == <>[]Done
=============================================================================