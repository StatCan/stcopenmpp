<template>
<div class="fullscreen text-grey-8 text-center q-pa-md flex flex-center">
  <div>
    <div class="text-h1">
      404
    </div>

    <div class="text-h2" style="opacity:.5">
      {{ $t('Oops. Nothing here...') }}
    </div>

    <q-btn
      class="q-mt-xl"
        color="white"
        text-color="blue"
      unelevated
      to="/"
      :label="$t('Go to Models List')"
      no-caps
    />
  </div>
</div>
</template>

<script>
export default {
  name: 'NotFound404'
}
</script>
