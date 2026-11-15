import './style.css'

const app = document.querySelector<HTMLDivElement>('#app')
if (!app) throw new Error('Application root is missing')

app.innerHTML = `
  <main class="shell">
    <header class="masthead"><span class="mark" aria-hidden="true">PP</span> Podium Picks</header>
    <section aria-labelledby="welcome">
      <p class="eyebrow">Formula 1 · A league of friends</p>
      <h1 id="welcome">Your three picks.<br>Your place on the podium.</h1>
      <p class="intro">Pick P1, P2, and P3. Lock them in before race time,
        then see who called it best.</p>
      <p class="status">The league is taking shape. Invitations and sign-in are coming next.</p>
    </section>
    <section class="rules" aria-labelledby="scoring">
      <h2 id="scoring">Every position counts.</h2>
      <dl>
        <div><dt>Exact position</dt><dd>5 <span>points</span></dd></div>
        <div><dt>Driver on the podium</dt><dd>2 <span>points in a different position</span></dd></div>
        <div><dt>Outside the podium</dt><dd>0 <span>points</span></dd></div>
      </dl>
      <p>Each pick earns one score. Points never stack.</p>
    </section>
    <footer>A private prediction league. Built for race weekends.</footer>
  </main>
`
