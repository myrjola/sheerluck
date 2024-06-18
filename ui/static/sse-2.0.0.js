// BSD 2-Clause License | Copyright (c) 2020, Big Sky Software All rights reserved.
!function(){var e;function t(e){return new EventSource(e,{withCredentials:!0})}function n(t,i){if(null==t)return null;s(t,"sse-connect").forEach((function(t){var s=e.getAttributeValue(t,"sse-connect");null!=s&&function(t,s,a){var o=htmx.createEventSource(s);o.onerror=function(s){if(e.triggerErrorEvent(t,"htmx:sseError",{error:s,source:o}),!r(t)&&o.readyState===EventSource.CLOSED){a=a||0;var i=Math.random()*(2^a)*500;window.setTimeout((function(){n(t,Math.min(7,a+1))}),i)}},o.onopen=function(n){e.triggerEvent(t,"htmx:sseOpen",{source:o})},e.getInternalData(t).sseEventSource=o}(t,s,i)})),function(t){s(t,"sse-swap").forEach((function(n){var s=e.getClosestMatch(n,o);if(null==s)return null;for(var i=e.getInternalData(s).sseEventSource,u=e.getAttributeValue(n,"sse-swap").split(","),c=0;c<u.length;c++){var l=u[c].trim(),v=function(o){r(s)||(e.bodyContains(n)?e.triggerEvent(t,"htmx:sseBeforeMessage",o)&&(a(n,o.data),e.triggerEvent(t,"htmx:sseMessage",o)):i.removeEventListener(l,v))};e.getInternalData(n).sseEventListener=v,i.addEventListener(l,v)}})),s(t,"hx-trigger").forEach((function(n){var s=e.getClosestMatch(n,o);if(null==s)return null;var a=e.getInternalData(s).sseEventSource,i=e.getAttributeValue(n,"hx-trigger");if(null!=i&&"sse:"==i.slice(0,4)){var u=function(t){r(s)||(e.bodyContains(n)||a.removeEventListener(i,u),htmx.trigger(n,i,t),htmx.trigger(n,"htmx:sseMessage",t))};e.getInternalData(t).sseEventListener=u,a.addEventListener(i.slice(4),u)}}))}(t)}function r(t){if(!e.bodyContains(t)){var n=e.getInternalData(t).sseEventSource;if(null!=n)return n.close(),!0}return!1}function s(t,n){var r=[];return e.hasAttribute(t,n)&&r.push(t),t.querySelectorAll("["+n+"], [data-"+n+"]").forEach((function(e){r.push(e)})),r}function a(t,n){e.withExtensions(t,(function(e){n=e.transformResponse(n,null,t)}));var r=e.getSwapSpecification(t),s=e.getTarget(t);e.swap(s,n,r)}function o(t){return null!=e.getInternalData(t).sseEventSource}htmx.defineExtension("sse",{init:function(n){e=n,null==htmx.createEventSource&&(htmx.createEventSource=t)},onEvent:function(t,r){var s=r.target||r.detail.elt;switch(t){case"htmx:beforeCleanupElement":var a=e.getInternalData(s);return void(a.sseEventSource&&a.sseEventSource.close());case"htmx:afterProcessNode":n(s)}}})}();