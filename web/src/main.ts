import "./style.css";
import { completeCallback, isSupportedLocalOrigin, loadConfig, makeAuth, signOut } from "./auth";
import { leagueRequest, type League } from "./api";
import type { User } from "oidc-client-ts";

const app = document.querySelector<HTMLDivElement>("#app");
if (!app) throw new Error("Application root is missing");
const root = app;
const config = loadConfig(import.meta.env);
const auth = config ? makeAuth(config) : null;
let user: User | null = null;

root.innerHTML = `
  <div class="app-shell">
    <header class="topbar"><a class="brand" href="/"><span class="brand-mark">P<span>P</span></span><span>PODIUM<span class="brand-accent"> PICKS</span></span></a><div class="topbar-right"><span class="season-label">THE PADDOCK <span> / </span> ${new Date().getFullYear()}</span><button class="button button-small button-outline" id="auth-button" type="button">Sign in</button></div></header>
    <main>
      <section class="hero"><div class="hero-copy"><p class="kicker"><span class="kicker-line"></span> YOUR RACE WEEKEND, SORTED</p><h1>Good calls.<br><em>Great rivalries.</em></h1><p class="hero-description">Three picks. One podium. A whole season of bragging rights with your people.</p><div class="hero-actions"><button class="button button-primary" id="hero-button" type="button">Get started</button><a class="text-link" href="#how-it-works">See how it works <span aria-hidden="true">↓</span></a></div><div class="hero-footnote"><span class="signal"></span> MADE FOR THE FRIENDS WHO NEVER MISS A RACE</div></div><div class="podium-visual" aria-hidden="true"><div class="visual-top"><span>RACE DAY</span><span>THE GRID / 01</span></div><div class="visual-label">YOUR CALL.<br><strong>YOUR PODIUM.</strong></div><div class="podium"><div class="step second"><span>02</span></div><div class="step first"><span>01</span><i>★</i></div><div class="step third"><span>03</span></div></div><div class="visual-bottom"><span>P1 · P2 · P3</span><span>EVERY PICK COUNTS</span></div></div></section>
      <section class="workspace" id="paddock"><div class="section-heading"><div><p class="kicker"><span class="kicker-line"></span> YOUR PADDOCK</p><h2>Make it official.</h2><p>Sign in to create your league or find one you already belong to.</p></div><span class="section-tag">LEAGUE CONTROL</span></div><div class="workspace-grid"><article class="workspace-card"><div class="card-top"><span class="card-icon">✳</span><span class="card-index">FOR THE ORGANIZER</span></div><h3>Start a league</h3><p>Give your crew a name and a place to compete.</p><form id="create-form"><label for="league-id">League ID</label><input id="league-id" name="leagueId" placeholder="e.g. sunday-grid" required maxlength="128" pattern="[A-Za-z0-9_-]+"><small>Letters, numbers, hyphens and underscores.</small><label for="league-name">League name</label><input id="league-name" name="name" placeholder="e.g. The Sunday Grid" required maxlength="200"><label for="display-name">Your display name</label><input id="display-name" name="displayName" placeholder="What your friends call you" required maxlength="160"><button class="button button-primary full" type="submit">Create league</button></form></article><article class="workspace-card join-card"><div class="card-top"><span class="card-icon card-icon-muted">⌕</span><span class="card-index">ALREADY A MEMBER?</span></div><h3>Find your league</h3><p>Enter a league ID to see the league details.</p><form id="find-form"><label for="find-id">League ID</label><input id="find-id" name="leagueId" placeholder="Your league ID" required maxlength="128"><button class="button button-dark full" type="submit">Find league</button></form><div class="league-result" id="league-result" hidden><span>YOUR LEAGUE</span><strong id="result-name"></strong><small id="result-id"></small></div><div class="note"><span>✳</span><p>Invitations and race picks are on their way. For now, organizers can create a league and members can look it up.</p></div></article></div><p id="message" class="message" role="status" aria-live="polite"></p></section>
      <section class="how" id="how-it-works"><div><p class="kicker"><span class="kicker-line"></span> THE FORMAT</p><h2>A little prediction.<br><em>A lot on the line.</em></h2></div><div class="rules-list"><div><span class="rule-icon">01</span><div><h3>Make your picks</h3><p>Choose your top three before the lights go out.</p></div></div><div><span class="rule-icon">02</span><div><h3>Watch it unfold</h3><p>See how your predictions stack up on race day.</p></div></div><div><span class="rule-icon">03</span><div><h3>Take the points</h3><p>5 for an exact match. 2 for a driver in another podium spot.</p></div></div></div></section>
    </main><footer><a class="brand" href="/"><span class="brand-mark">P<span>P</span></span><span>PODIUM<span class="brand-accent"> PICKS</span></span></a><span>BUILT FOR RACE WEEKENDS & GOOD COMPANY.</span><span>SEE YOU AT THE FINISH LINE.</span></footer>
  </div>`;

const button = document.querySelector<HTMLButtonElement>("#auth-button")!;
const heroButton = document.querySelector<HTMLButtonElement>("#hero-button")!;
const message = document.querySelector<HTMLParagraphElement>("#message")!;
const createForm = document.querySelector<HTMLFormElement>("#create-form")!;
const findForm = document.querySelector<HTMLFormElement>("#find-form")!;
const showMessage = (text: string, error = false) => {
  message.textContent = text;
  message.classList.toggle("error", error);
};

function renderSession() {
  const signedIn = !!user && !user.expired;
  button.textContent = signedIn ? "Sign out" : "Sign in";
  heroButton.textContent = signedIn ? "Go to your paddock" : "Get started";
  createForm.querySelectorAll("input, button").forEach((element) => {
    (element as HTMLInputElement).disabled = !signedIn;
  });
  findForm.querySelectorAll("input, button").forEach((element) => {
    (element as HTMLInputElement).disabled = !signedIn;
  });
}

async function beginSignIn() {
  if (import.meta.env.DEV && !isSupportedLocalOrigin(window.location.origin)) {
    showMessage("For local sign-in, open http://localhost:5173/ and try again.", true);
    message.scrollIntoView({ behavior: "smooth", block: "center" });
    return;
  }
  if (!auth) {
    showMessage(
      "Sign-in is not configured yet. See the web setup instructions.",
      true,
    );
    return;
  }
  try {
    await auth.signinRedirect();
  } catch {
    showMessage("Could not start sign-in. Please try again.", true);
  }
}
button.addEventListener("click", () => {
  if (user && auth && config) void signOut(auth, config);
  else void beginSignIn();
});
heroButton.addEventListener("click", () => {
  if (user && !user.expired)
    document.querySelector("#paddock")?.scrollIntoView({ behavior: "smooth" });
  else void beginSignIn();
});

createForm.addEventListener("submit", (event) => {
  event.preventDefault();
  if (!config || !user || user.expired) {
    showMessage("Please sign in again.", true);
    renderSession();
    return;
  }
  const data = new FormData(createForm);
  const request = {
    leagueId: String(data.get("leagueId")),
    name: String(data.get("name")).trim(),
    displayName: String(data.get("displayName")).trim(),
  };
  if (
    !request.name ||
    !request.displayName ||
    Array.from(request.displayName).length > 80 ||
    Array.from(request.name).length > 200
  ) {
    showMessage(
      "Enter a league name and a display name of at most 80 characters.",
      true,
    );
    return;
  }
  showMessage("Creating your league…");
  const submit = createForm.querySelector<HTMLButtonElement>(
    'button[type="submit"]',
  )!;
  submit.disabled = true;
  void leagueRequest<{ leagueId: string }>(
    config,
    user.access_token,
    "/leagues",
    { method: "POST", body: JSON.stringify(request) },
  )
    .then((league) => {
      showMessage(`League “${league.leagueId}” created.`);
      createForm.reset();
    })
    .catch((error: unknown) =>
      showMessage(
        error instanceof Error
          ? error.message
          : "Could not connect. Please try again.",
        true,
      ),
    )
    .finally(() => {
      submit.disabled = false;
    });
});
findForm.addEventListener("submit", (event) => {
  event.preventDefault();
  if (!config || !user || user.expired) {
    showMessage("Please sign in again.", true);
    renderSession();
    return;
  }
  const leagueId = String(new FormData(findForm).get("leagueId")).trim();
  if (!leagueId) return;
  const result = document.querySelector<HTMLElement>("#league-result")!;
  result.hidden = true;
  showMessage("Looking up your league…");
  void leagueRequest<League>(
    config,
    user.access_token,
    `/leagues/${encodeURIComponent(leagueId)}`,
  )
    .then((league) => {
      document.querySelector("#result-name")!.textContent = league.name;
      document.querySelector("#result-id")!.textContent = league.leagueId;
      result.hidden = false;
      showMessage("League found.");
    })
    .catch((error: unknown) =>
      showMessage(
        error instanceof Error
          ? error.message
          : "Could not connect. Please try again.",
        true,
      ),
    );
});

renderSession();
if (window.location.pathname === "/auth/callback") {
  if (auth)
    void completeCallback(auth)
      .then((signedIn) => {
        user = signedIn;
        renderSession();
        showMessage("You’re signed in. Welcome to the paddock.");
      })
      .catch(() =>
        showMessage("Sign-in could not be verified. Please try again.", true),
      );
  else {
    window.history.replaceState({}, document.title, "/");
    showMessage("Sign-in is not configured yet.", true);
  }
} else if (!config)
  showMessage(
    "Set up your public Cognito and API configuration to enable sign-in.",
    true,
  );
