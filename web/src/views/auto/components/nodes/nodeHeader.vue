<template>
  <div class="header">{{ title }}</div>
</template>
<script>
  import { defineComponent } from 'vue';

  export default defineComponent({
    props: ['title'],
  });
</script>

<style scoped>
  div {
    background: #494949;
    padding: 10px 15px;
    margin: -15px -15px 15px;
  }
</style>
