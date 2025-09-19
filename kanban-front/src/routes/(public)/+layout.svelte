<script lang="ts">
  let { children } = $props();
</script>

<div class="min-h-screen bg-gray-100 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
  {@render children()}
</div>