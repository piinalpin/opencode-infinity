<script lang="ts">
  let count = $state(0);
</script>
<!-- shortcut: UI sintetis tanpa state pool; dashboard dan polling mengikuti EP-05. -->
<main>
  <h1>Fixture kemasan T05</h1>
  <p>Data sintetis. Bukan dashboard pemantauan pool.</p>
  <button onclick={() => count += 1}>Uji interaksi lokal</button>
  <p role="status">Interaksi lokal: {count}</p>
</main>
<style>
  :global(body) { margin: 0; padding: 2rem; font-family: system-ui, sans-serif; background: #f4f6fa; color: #172033; }
  main { max-width: 640px; margin: 0 auto; }
  button { font: inherit; padding: .6rem 1rem; }
</style>
